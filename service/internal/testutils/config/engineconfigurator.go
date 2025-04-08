package config_test

type TestEngineConfigurator struct{}

func (cfg TestEngineConfigurator) GetVersion() (string, error) {
	return "", nil
}

func NewTestVersionConfigurator() TestEngineConfigurator {
	return TestEngineConfigurator{}
}
