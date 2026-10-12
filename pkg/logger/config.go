package logger

type Config struct {
	Level  string `koanf:"level" validate:"required,oneof=trace debug info warn error fatal panic"`
	Caller int    `koanf:"caller" validate:"min=0"` // whether to add source code information (file and line number) to log entries. 0 disables this feature, and any positive integer enables it with the specified call depth.
}

var defaultCfg = Config{
	Level:  "info",
	Caller: 0,
}
