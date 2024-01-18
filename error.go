package natsmicromw

type HandlerError struct {
	Description string `json:"description"`
	Code        string `json:"code"`
}

func (e *HandlerError) Error() string {
	return e.Description
}
