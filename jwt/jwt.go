package jwt

import (
	"errors"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/dgrijalva/jwt-go"

	"github.com/satriahrh/letter-block/data"
)

// secret key being used to sign tokens
var (
	privateKey, publicKey = []byte{}, []byte{}
)

type User struct {
	PlayerId          data.PlayerId
	DeviceFingerprint data.DeviceFingerprint
	ExpiredAt         int64
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

// GenerateToken generates a jwt token and assign a username to it's claims and return it
func GenerateToken(player data.Player) (string, error) {
	token := jwt.New(jwt.SigningMethodRS256)
	/* Create a map to store our claims */
	claims := token.Claims.(jwt.MapClaims)
	/* Set token claims */
	claims["playerId"] = strconv.FormatUint(uint64(player.Id), 32)
	claims["deviceFingerprint"] = player.DeviceFingerprint
	claims["expiredAt"] = strconv.FormatInt(player.SessionExpiredAt, 10)

	if len(privateKey) == 0 {
		privateKey = []byte(os.Getenv("RSA_PRIVATE_KEY"))
	}
	rsaPrivateKey, err := jwt.ParseRSAPrivateKeyFromPEM(privateKey)
	if err != nil {
		log.Println(err)
		return "", err
	}
	tokenString, err := token.SignedString(rsaPrivateKey)
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	return tokenString, nil
}

// ParseToken parses a jwt token and returns the username it it's claims
func ParseToken(tokenStr string) (User, error) {
	token, err := jwt.Parse(tokenStr, keyFunc)
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		expiredAt := claims["expiredAt"]
		if expiredAt == nil {
			log.Println("expiredAt is nil")
			return User{}, errors.New("invalid token")
		}

		expiredAtUnix, err := strconv.ParseInt(expiredAt.(string), 10, 64)
		if err != nil {
			log.Println(err)
			return User{}, err
		}

		if time.Unix(expiredAtUnix, 0).After(time.Now()) {
			log.Println("token expired")
			return User{}, errors.New("expired token")
		}

		var user User
		playerId := claims["playerId"]
		deviceFingerprint := claims["deviceFingerprint"]

		playerIdUint64, err := strconv.ParseUint(playerId.(string), 32, 64)
		if err != nil {
			log.Println(err)
			return User{}, err
		}
		user.PlayerId = data.PlayerId(playerIdUint64)
		user.DeviceFingerprint = data.DeviceFingerprint(deviceFingerprint.(string))

		return user, nil
	} else {
		log.Println(err)
		return User{}, errors.New("invalid token")
	}
}
