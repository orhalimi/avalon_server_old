package main

import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type User struct {
	Id       string `json:"id"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type UserService interface {
	CreateUser(u *User) error
	GetByUsername(username string) (*User, error)
}

type userModel struct {
	Id       bson.ObjectId `bson:"_id,omitempty"`
	Username string
	Password string
}

func userModelIndex() mgo.Index {
	return mgo.Index{
		Key:        []string{"username"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	}
}

func newUserModel(u *User) *userModel {
	return &userModel{
		Username: u.Username,
		Password: u.Password}
}

func (u *userModel) toRootUser() *User {
	return &User{
		Id:       u.Id.Hex(),
		Username: u.Username,
		Password: u.Password}
}

type UserService1 struct {
	collection *mgo.Collection
	hash       HashAbs
}

func NewUserService(session *Session, dbName string, collectionName string, h HashAbs) *UserService1 {
	collection := session.GetCollection(dbName, collectionName)
	collection.EnsureIndex(userModelIndex())
	return &UserService1{collection, h}
}

func (p *UserService1) Create(u *User) error {
	user := newUserModel(u)
	hashedPassword, err := p.hash.Generate(user.Password)
	if err != nil {
		return err
	}
	user.Password = hashedPassword
	return p.collection.Insert(&user)
}

func (p *UserService1) GetByUsername(username string) (*User, error) {
	model := userModel{}
	err := p.collection.Find(bson.M{"username": username}).One(&model)
	return model.toRootUser(), err
}
