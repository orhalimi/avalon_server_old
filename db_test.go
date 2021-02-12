package main

import (
	"log"
	"testing"
)


type MockHash struct{}

func (h *MockHash) Generate(s string) (string, error) {
	return s, nil
}

func (h *MockHash) Compare(hash string, s string) error {
	return nil
}


func Test_UserService(t *testing.T) {
	t.Run("CreateUser", createUser_should_insert_user_into_mongo)
}

func createUser_should_insert_user_into_mongo(t *testing.T) {
	//Arrange
	session, err := NewSession(mongoUrl)
	if(err != nil) {
		log.Fatalf("Unable to connect to mongo: %s", err)
	}
	defer func() {
		session.DropDatabase(dbName)
		session.Close()
	}()

	mockHash := MockHash{}
	userService := NewUserService(session.Copy(), dbName, userCollectionName, &mockHash)

	testUsername := "integration_test_user"
	testPassword := "integration_test_password"
	user := User{
		Username: testUsername,
		Password: testPassword }

	//Act
	err = userService.Create(&user)

	//Assert
	if err != nil {
		t.Error("Unable to create user:", err)
	}
	var results []User
	session.GetCollection(dbName,userCollectionName).Find(nil).All(&results)

	count := len(results)
	if(count != 1) {
		t.Error("Incorrect number of results. Expected `1`, got: `%i`", count)
	}
	if(results[0].Username != user.Username) {
		t.Error("Incorrect Username. Expected, Got: ", testUsername, results[0].Username)
	}
}
