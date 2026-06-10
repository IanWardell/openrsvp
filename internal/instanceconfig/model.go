package instanceconfig

// Known instance_config keys. These hold only non-secret instance settings;
// secrets (SMTP passwords, API keys, Twilio tokens) MUST stay in env.
const (
	KeyInstanceName    = "instance_name"
	KeyDefaultTimezone = "default_timezone"
	KeyAllowSignups    = "allow_signups"
	KeySupportEmail    = "support_email"
	KeyConfigured      = "configured"
)

// Settings is the typed view of the editable instance settings.
type Settings struct {
	InstanceName    string `json:"instanceName"`
	DefaultTimezone string `json:"defaultTimezone"`
	AllowSignups    bool   `json:"allowSignups"`
	SupportEmail    string `json:"supportEmail"`
	Configured      bool   `json:"configured"`
}

// PublicConfig is the non-sensitive subset exposed to the SPA for unauthenticated
// or general use. It deliberately omits the "configured" flag detail beyond what
// the status endpoint already reports.
type PublicConfig struct {
	InstanceName    string `json:"instanceName"`
	DefaultTimezone string `json:"defaultTimezone"`
	AllowSignups    bool   `json:"allowSignups"`
	SupportEmail    string `json:"supportEmail"`
}
