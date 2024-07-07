package config_test

type TestVersionConfiguratorFunc func() (string, error)

func (f TestVersionConfiguratorFunc) GetVersion() (string, error) {
	return f()
}

func NewTestVersionConfigurator() TestVersionConfiguratorFunc {
	return func() (string, error) { return "", nil }
}
