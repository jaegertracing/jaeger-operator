package v1

const (
	// LabelManagedBy is used as the key to the label indicating that this instance is managed by an operator
	LabelManagedBy string = "app.kubernetes.io/managed-by"

	// ConfigIdentity is the key to the configuration map related to the operator's identity
	ConfigIdentity string = "identity"
)
