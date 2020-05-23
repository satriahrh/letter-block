package auth

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/satriahrh/letter-block/data"
	"github.com/satriahrh/letter-block/jwt"
)

var userCtxKey = &contextKey{"user"}

type contextKey struct {
	name string
}

type Authentication struct {
	transactional data.Transactional
}

func New(transactional data.Transactional) *Authentication {
	return &Authentication{transactional}
}

func (a *Authentication) Authenticate(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(0)
	if err != nil {
		log.Println(err)
		errorResponse(w, 403, "cannot authenticate you")
		return
	}

	rawDeviceFingerprint := r.Form.Get("deviceFingerprint")
	deviceFingerprint := data.DeviceFingerprint(rawDeviceFingerprint)
	player, err := authentication(r.Context(), a, deviceFingerprint)
	if err != nil {
		log.Println(err)
		errorResponse(w, 500, "cannot generating token")
		return
	}

	token, err := jwt.GenerateToken(player)
	if err != nil {
		log.Println(err)
		errorResponse(w, 500, "cannot generating token")
		return
	}

	successResponse(w, struct {
		Token     string `json:"token"`
		ExpiredIn int64  `json:"expired_in"`
	}{
		token,
		player.SessionExpiredAt - time.Now().Unix(),
	})
}

func (a *Authentication) Register(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(0)
	if err != nil {
		log.Println(err)
		errorResponse(w, 403, "cannot register you")
		return
	}

	rawDeviceFingerprint := r.Form.Get("deviceFingerprint")
	rawUsername := r.Form.Get("username")
	deviceFingerprint := data.DeviceFingerprint(rawDeviceFingerprint)
	err = func() (e error) {
		tx, e := a.transactional.BeginTransaction(r.Context())
		if e != nil {
			log.Println(e)
			return
		}
		defer func() {
			e = a.transactional.FinalizeTransaction(tx, e)
		}()

		e = a.transactional.UpsertPlayer(r.Context(), tx, data.Player{
			Username:          rawUsername,
			DeviceFingerprint: deviceFingerprint,
		})

		return
	}()
	if err != nil {
		log.Println(err)
		errorResponse(w, 500, "cannot generating token")
	}

	successResponse(w, "success")
}

// HttpMiddleware will authenticate from the token
func (a *Authentication) HttpMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := func() string {
			authorization := r.Header.Get("Authorization")
			if len(authorization) == 0 {
				authorization = r.URL.Query().Get("Authorization")
			}
			if len(authorization) < 7 {
				return ""
			}
			return authorization[7:]
		}()

		// validate jwt token
		user, err := jwt.ParseToken(token)
		if user.PlayerId == 0 || err != nil {
			errorResponse(w, http.StatusForbidden, "invalid token")
			return
		}

		// put it in context
		ctx := context.WithValue(r.Context(), userCtxKey, user)

		// and call the next with our new context
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

// ForContext finds the user from the context. REQUIRES Middleware to have run.
func ForContext(ctx context.Context) jwt.User {
	raw, _ := ctx.Value(userCtxKey).(jwt.User)
	return raw
}

func authentication(ctx context.Context, a *Authentication, fingerprint data.DeviceFingerprint) (player data.Player, err error) {
	tx, err := a.transactional.BeginTransaction(ctx)
	if err != nil {
		log.Println(err)
		return
	}
	defer func() {
		err = a.transactional.FinalizeTransaction(tx, err)
	}()

	player, err = a.transactional.GetPlayerByDeviceFingerprint(ctx, tx, fingerprint)
	if err != nil {
		log.Println(err)
		return
	}

	currentTime := time.Now()
	if player.SessionExpiredAt < currentTime.Unix()-10 {
		player.SessionExpiredAt = currentTime.Add(1 * time.Minute).Unix()
		err = a.transactional.UpsertPlayer(ctx, tx, player)
		if err != nil {
			log.Println(err)
		}
	}

	return
}

func errorResponse(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(
		struct {
			Message string `json:"message"`
		}{message},
	)
}

func successResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	_ = json.NewEncoder(w).Encode(
		struct {
			Data interface{} `json:"data"`
		}{data},
	)
}
