// Example compression middleware for natsmicromw

package middleware

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"errors"
	"io"

	"github.com/Karimerto/natsmicromw"
)

type CompressionType string

const (
	CompressionNone    CompressionType = ""
	CompressionGzip    CompressionType = "gzip"
	CompressionDeflate CompressionType = "deflate"

	HeaderAcceptEncoding string = "accept-encoding"
	HeaderEncoding              = "encoding"
)

var (
	// Do not compress messages smaller than this limit
	compressMin = 1000

	ErrUnsupportedEncoding = errors.New("unsupported encoding")
)

// SetCompressMin sets the global minimum size for compression.
func SetCompressMin(minBytes int) {
	compressMin = minBytes
}

// GetCompressMin retrieves the current global minimum size for compression.
func GetCompressMin() int {
	return compressMin
}

func compressGzip(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	if _, err := writer.Write(data); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func compressDeflate(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	writer, err := flate.NewWriter(&buf, flate.DefaultCompression)
	if err != nil {
		return nil, err
	}
	if _, err := writer.Write(data); err != nil {
		return nil, err
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// compressMessage compresses the message data if it exceeds the threshold.
func compressReply(compression CompressionType, reply *natsmicromw.MicroReply) error {
	if len(reply.Data) < GetCompressMin() {
		return nil
	}

	if compression != CompressionNone {
		switch compression {
		case CompressionGzip:
			compressedData, err := compressGzip(reply.Data)
			if err != nil {
				return err
			}
			reply.Data = compressedData
			reply.HeaderSet(HeaderEncoding, string(compression))

		case CompressionDeflate:
			compressedData, err := compressDeflate(reply.Data)
			if err != nil {
				return err
			}
			reply.Data = compressedData
			reply.HeaderSet(HeaderEncoding, string(compression))
		}
	}

	return nil
}

func decompressGzip(data []byte) ([]byte, error) {
	zr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	return io.ReadAll(zr)
}

func decompressDeflate(data []byte) ([]byte, error) {
	fr := flate.NewReader(bytes.NewReader(data))
	defer fr.Close()
	return io.ReadAll(fr)
}

// Read possibly-compressed content
func readCompressedData(encoding string, data []byte) ([]byte, error) {
	if encoding == "gzip" {
		return decompressGzip(data)
	} else if encoding == "deflate" {
		return decompressDeflate(data)
	} else if len(encoding) > 0 {
		return nil, ErrUnsupportedEncoding
	}
	return data, nil
}

// decompressRequest decompresses the message data if it was compressed.
func decompressRequest(req *natsmicromw.MicroRequest) error {
	data, err := readCompressedData(req.HeaderGet(HeaderEncoding), req.Data)
	if err != nil {
		return err
	}
	req.Data = data

	return nil
}

func CompressionMiddleware(next natsmicromw.MicroHandlerFunc) natsmicromw.MicroHandlerFunc {
	return func(req *natsmicromw.MicroRequest) (*natsmicromw.MicroReply, error) {
		// Decompress incoming request
		if err := decompressRequest(req); err != nil {
			return nil, err
		}

		// Call next function in the chain
		res, err := next(req)
		if err != nil {
			return nil, err
		}

		// Finally also compress reply
		accept := CompressionType(req.HeaderGet(HeaderAcceptEncoding))
		if err := compressReply(accept, res); err != nil {
			return nil, err
		}

		return res, nil
	}
}
