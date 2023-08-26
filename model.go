package mx

type Model[E any] interface {
	ToEntity() E
	FromEntity(entity E) interface{}
}
