package config_test

type TestEngineConfigurator struct{}

func (cfg TestEngineConfigurator) GetVersion() (string, error) {
	return "", nil
}

func (cfg TestEngineConfigurator) GetExpectedHost() *string {
	return nil
}

func (cfg TestEngineConfigurator) GetRequestsPerSecond() (*int, error) {
	return nil, nil
}

func NewTestVersionConfigurator() TestEngineConfigurator {
	return TestEngineConfigurator{}
}
