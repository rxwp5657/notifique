package mock_controllers

type MockedStorage struct {
	*MockDistributionListStorage
	*MockUserStorage
	*MockNotificationStorage
}

func NewMockedStorage(dls *MockDistributionListStorage, us *MockUserStorage,
	ns *MockNotificationStorage) *MockedStorage {

	return &MockedStorage{
		dls,
		us,
		ns,
	}
}
