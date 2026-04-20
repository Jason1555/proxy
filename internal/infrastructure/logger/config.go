package logger

type Config struct {
	Level            string         `json:"level" yaml:"level" toml:"level"`                                        //Уровень логгирования
	Encoding         string         `json:"encoding" yaml:"encoding" toml:"encoding"`                               //Формат логов
	OutputPaths      []string       `json:"output_paths" yaml:"output_paths" toml:"output_paths"`                   //Пути вывода логов
	ErrorOutputPaths []string       `json:"error_output_paths" yaml:"error_output_paths" toml:"error_output_paths"` //Пути вывода ошибок
	WithCaller       bool           `json:"with_caller" yaml:"with_caller" toml:"with_caller"`                      //Логировать ли информацию о месте вызова
	InitialFields    map[string]any `json:"initial_fields" yaml:"initial_fields" toml:"initial_fields"`             //Начальные поля
}

func NewDefaultConfig() *Config {
	return &Config{
		Level:            "info",
		Encoding:         "json",
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		WithCaller:       true,
		InitialFields: map[string]any{
			"service": "proxy-server",
		},
	}
}

func NewDevConfig() *Config {
	return &Config{
		Level:            "debug",
		Encoding:         "console",
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		WithCaller:       true,
		InitialFields: map[string]any{
			"service": "proxy-server",
			"env":     "dev",
		},
	}
}

func NewProdConfig() *Config {
	return &Config{
		Level:            "info",
		Encoding:         "json",
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		WithCaller:       true,
		InitialFields: map[string]any{
			"service": "proxy-server",
			"env":     "prod",
		},
	}
}
