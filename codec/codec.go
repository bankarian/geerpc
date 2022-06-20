package codec

type Header struct {
	ServiceMethod string
	Seq           uint64 // sequence number chosen by client
	Error         string
}
