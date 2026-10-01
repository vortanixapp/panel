package handlers

import "golang.org/x/crypto/bcrypt"

var timingEqualizerHash, _ = bcrypt.GenerateFromPassword([]byte("vortanix-timing-equalizer"), bcrypt.DefaultCost)

func burnPasswordCheck(password string) {
	_ = bcrypt.CompareHashAndPassword(timingEqualizerHash, []byte(password))
}
