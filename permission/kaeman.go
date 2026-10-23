package permission

type Permission uint64

const (
	BaseDeveloper Permission = 1 << iota // base:developer
	BaseDefault                          // base:default

	SplasherQueue // splasher:queue
)
