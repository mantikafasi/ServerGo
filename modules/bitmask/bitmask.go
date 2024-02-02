package bitmask

const (
	UserAdmin = 1 << iota
	UserModerator
	UserDonor
	UserBanned
)


func CheckFlag(user int32, permission int32) bool {
	return user & permission != 0
}

func SetFlag(user int32, permission int32) int32 {
	return user | permission
}	