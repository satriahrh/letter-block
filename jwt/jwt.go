package jwt

import (
	"log"
	"os"
	"strconv"

	"github.com/dgrijalva/jwt-go"

	"github.com/satriahrh/letter-block/data"
)

// secret key being used to sign tokens
var (
	privateKey, publicKey = []byte{}, []byte{}
)

type User struct {
	PlayerId data.PlayerId
	DeviceId uint64
}

func keyFunc(token *jwt.Token) (interface{}, error) {
	if len(publicKey) == 0 {
		publicKey = []byte(os.Getenv("RSA_PUBLIC_KEY"))
	}

	key, err := jwt.ParseRSAPublicKeyFromPEM(publicKey)
	if err != nil {
		return nil, err
	}
	return key, nil
}

// // GenerateToken generates a jwt token and assign a username to it's claims and return it
// func GenerateToken(username string) (string, error) {
// 	token := jwt.New(jwt.SigningMethodRS256)
// 	/* Create a map to store our claims */
// 	claims := token.Claims.(jwt.MapClaims)
// 	/* Set token claims */
// 	claims["username"] = username
// 	claims["exp"] = time.Now().Add(time.Hour * 24).Unix()
//
// 	if len(privateKey) == 0 {
// 		privateKey = []byte(os.Getenv("RSA_PRIVATE_KEY"))
// 	}
// 	tokenString, err := token.SignedString(privateKey)
// 	if err != nil {
// 		log.Fatal("Error in Generating key")
// 		return "", err
// 	}
// 	return tokenString, nil
// }

// ParseToken parses a jwt token and returns the username it it's claims
func ParseToken(tokenStr string) (User, error) {
	token, err := jwt.Parse(tokenStr, keyFunc)
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		var user User
		playerId := claims["playerId"]
		deviceId := claims["deviceId"]

		playerIdUint64, err := strconv.ParseUint(playerId.(string), 10, 64)
		if err != nil {
			log.Println(err)
			return User{}, err
		}
		user.PlayerId = data.PlayerId(playerIdUint64)

		user.DeviceId, err = strconv.ParseUint(deviceId.(string), 10, 64)
		if err != nil {
			log.Println(err)
			return User{}, err
		}

		return user, nil
	} else {
		log.Println(err)
		return User{}, err
	}
}
