package main

import (
    "errors"
    "net/http"
    "os"

    "github.com/gin-gonic/gin"
    "github.com/jinzhu/gorm"
    _ "github.com/jinzhu/gorm/dialects/mysql"
    "github.com/dgrijalva/jwt-go"
    "github.com/asaskevich/govalidator"
    "golang.org/x/crypto/bcrypt"
)

var (
    db *gorm.DB
)

func main() {
  
    r := gin.Default()

    connectDB()

    
    r.POST("/user/register", registerUser)
    r.POST("/user/login", loginUser)
    r.PUT("/user/:userId", updateUser)
    r.DELETE("/user/:userId", deleteUser)

    r.POST("/photo", authMiddleware(), uploadPhoto)
    r.GET("/photo", getPhoto)
    r.PUT("/photo/:photoId", authMiddleware(), updatePhoto)
    r.DELETE("/photo/:photoId", authMiddleware(), deletePhoto)

  
    r.Run(":8080")
}


func connectDB() {
    var err error
    db, err = gorm.Open("mysql", "root:@tcp(localhost:3306)/api?parseTime=true")
    if err != nil {
        panic("failed to connect database")
    }

    if err := db.AutoMigrate(&user{}, &Photo{}).Error; err != nil {
        panic("failed to auto migrate database")
    }
}


func authMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        tokenString := c.Request.Header.Get("Authorization")
        if tokenString == "" {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
            c.Abort()
            return
        }

        secretKey := os.Getenv("JWT_SECRET_KEY")
        if secretKey == "" {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
            c.Abort()
            return
        }

        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            return []byte(secretKey), nil
        })

        if err != nil || !token.Valid {
            c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
            c.Abort()
            return
        }

        c.Next()
    }
}


type user struct {
    gorm.Model
    Username string `json:"username" gorm:"not null"`
    Email    string `json:"email" gorm:"unique;not null"`
    Password string `json:"password" gorm:"not null"`
    Photo   []Photo
}

func (u *user) Validate() error {
    if !govalidator.IsEmail(u.Email) {
        return errors.New("Invalid email format")
    }

    if len(u.Password) < 6 {
        return errors.New("Password must be at least 6 characters long")
    }

    return nil
}

func (u *user) SetPassword(password string) error {
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return err
    }
    u.Password = string(hashedPassword)
    return nil
}


type Photo struct {
    gorm.Model
    Title    string `json:"title"`
    Caption  string `json:"caption"`
    PhotoURL string `json:"photo_url"`
    UserID   uint   `json:"user_id"`
}


func registerUser(c *gin.Context) {
    var newUser user
    if err := c.ShouldBindJSON(&newUser); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    if err := newUser.Validate(); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    if err := newUser.SetPassword(newUser.Password); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set password"})
        return
    }

    if err := db.Create(&newUser).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "User registered successfully"})
}

func loginUser(c *gin.Context) {
    var loginDetails user
    if err := c.ShouldBindJSON(&loginDetails); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
        return
    }

    var user user
    if err := db.Where("email = ?", loginDetails.Email).First(&user).Error; err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
        return
    }

    if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(loginDetails.Password)); err != nil {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid password"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Login successful"})
}

func updateUser(c *gin.Context) {
    userId := c.Param("userId")
    var updatedUser user
    if err := c.ShouldBindJSON(&updatedUser); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    if err := db.Model(&user{}).Where("id = ?", userId).Updates(updatedUser).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}

func deleteUser(c *gin.Context) {
    userId := c.Param("userId")
    if err := db.Where("id = ?", userId).Delete(&user{}).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

func uploadPhoto(c *gin.Context) {
    var newPhoto Photo
    if err := c.ShouldBindJSON(&newPhoto); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    if err := db.Create(&newPhoto).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload photo"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Photo uploaded successfully"})
}

func getPhoto(c *gin.Context) {
    var photo []Photo
    if err := db.Find(&photo).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve photo"})
        return
    }

    c.JSON(http.StatusOK, photo)
}

func updatePhoto(c *gin.Context) {
    photoId := c.Param("photoId")
    var updatedPhoto Photo
    if err := c.ShouldBindJSON(&updatedPhoto); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    if err := db.Model(&Photo{}).Where("id = ?", photoId).Updates(updatedPhoto).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update photo"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Photo updated successfully"})
}

func deletePhoto(c *gin.Context) {
    photoId := c.Param("photoId")
    if err := db.Where("id = ?", photoId).Delete(&Photo{}).Error; err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete photo"})
        return
    }

    c.JSON(http.StatusOK, gin.H{"message": "Photo deleted successfully"})
}
