package v1beta1

type AvailabilityStatus string

const (
	Available   AvailabilityStatus = "Available"
	Unavailable                    = "Unavailable"
	Pending                        = "Pending"
)
