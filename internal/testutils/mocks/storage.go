package mocks

type MockedRegistry struct {
	*MockDistributionRegistry
	*MockUserRegistry
	*MockNotificationRegistry
	*MockNotificationTemplateRegistry
}

func NewMockedRegistry(dlr *MockDistributionRegistry, ur *MockUserRegistry,
	nr *MockNotificationRegistry, ntr *MockNotificationTemplateRegistry) *MockedRegistry {

	return &MockedRegistry{
		dlr,
		ur,
		nr,
		ntr,
	}
}
