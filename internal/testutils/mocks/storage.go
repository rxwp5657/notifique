package mock_controllers

type MockedRegistry struct {
	*MockDistributionRegistry
	*MockUserRegistry
	*MockNotificationRegistry
}

func NewMockedRegistry(dls *MockDistributionRegistry, us *MockUserRegistry,
	ns *MockNotificationRegistry) *MockedRegistry {

	return &MockedRegistry{
		dls,
		us,
		ns,
	}
}
