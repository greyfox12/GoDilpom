package hash

import (
	"crypto/rand"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

// Bcript hash
func GetBcryptHash(text string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(text), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	//	return hex.EncodeToString(hash[:]), nil
	return string(hash), nil

}

// генерируем случайную последовательность байт
func generateRandom(size int) ([]byte, error) {
	b := make([]byte, size)
	_, err := rand.Read(b)
	if err != nil {
		return nil, err
	}

	return b, nil
}

type AuthGen struct {
	Secretkey []byte
}

func (h *AuthGen) GetSecretKey() []byte {
	return h.Secretkey
}

func (h *AuthGen) Init() error {
	var err error
	if h.Secretkey, err = generateRandom(32); err != nil {
		return fmt.Errorf("error generateRandom:  %w", err)
	}
	return nil
}

// Генерирую токен пользователя
type Claims struct {
	jwt.RegisteredClaims
	UserLogin string
}

func (h *AuthGen) CreateToken(login string) (string, error) {

	// создаём новый токен с алгоритмом подписи HS256 и утверждениями — Claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			// когда создан токен
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 72)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		// собственное утверждение
		UserLogin: login,
	})

	tokenString, err := token.SignedString(h.Secretkey)
	if err != nil {
		return "", err
	}

	// возвращаем строку токена
	return tokenString, nil
}

func (h *AuthGen) GetUserID(tokenString string) (string, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return h.Secretkey, nil
		})
	if err != nil {
		return "", err
	}

	if !token.Valid {
		return "", fmt.Errorf("generateRandom: token is not valid")
	}

	//	logmy.OutLogInfo(fmt.Errorf("generateRandom: token is valid, login: %v", claims.UserLogin))
	return claims.UserLogin, nil
}

func ValidLunaStr(vpan string) bool {

	x := 0
	s := 0
	vp := Reverse(vpan)
	for i, r := range strings.Split(vp, "") {
		x, _ = strconv.Atoi(r)
		//		fmt.Printf("x=%v i=%v\n", x, i)
		if i%2 != 0 {
			x = x * 2
			if x > 9 {
				x = x - 9
			}
		}
		s = s + x
		//		fmt.Printf("s=%v\n", s)
	}
	s = 10 - s%10
	if s == 10 {
		s = 0
	}
	return s == 0
}
func Reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

// Проверяю токен
func (h *AuthGen) CheckAuth(token string) (string, error) {
	if token == "" {
		return "", fmt.Errorf("checkauth: no autorization head")
	}

	tokenBuf := strings.Split(token, " ")
	if len(tokenBuf) != 2 {
		return "", fmt.Errorf("checkauth: unknow format autorization head: %v", token)
	}

	if tokenBuf[0] != "Bearer" {
		return "", fmt.Errorf("checkauth: unknow type autorization head: %v", tokenBuf[0])
	}

	login, err := h.GetUserID(tokenBuf[1])
	if err != nil {
		return "", fmt.Errorf("checkauth: fail getuserid from token: %w", err)
	}
	if login == "" {
		return "", fmt.Errorf("checkauth: fail getuserid from token: %v", tokenBuf[1])
	}

	return login, nil
}
