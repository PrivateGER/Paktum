package main

import (
	"Paktum/Database"
	"Paktum/graph"
	"Paktum/graph/generated"
	"context"
	"embed"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"io/fs"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

//go:embed paktum-fe/dist/*
var embeddedFrontend embed.FS

func getFrontendFS() http.FileSystem {
	fsys, err := fs.Sub(embeddedFrontend, "paktum-fe/dist")
	if err != nil {
		log.Fatal("Couldn't get frontend FS: %s", err)
		return nil
	}
	return http.FS(fsys)
}

func ServerMode(imageDir string) {

	rand.Seed(time.Now().UnixNano())
	r := gin.Default()

	r.Use(corsMiddleware)

	r.GET("/api/search", func(c *gin.Context) {
		query := c.Query("q")
		if query == "" {
			c.JSON(400, gin.H{
				"error": "No query provided",
			})
			return
		}
		limitString := c.Query("limit")
		if limitString == "" {
			limitString = "10"
		}
		limit, err := strconv.Atoi(limitString)
		if err != nil {
			c.JSON(400, gin.H{
				"error": "Invalid limit provided (0 < limit <= 50)",
			})
			return
		}

		images, resultCount, err := Database.SearchImages(query, limit, true, "")
		if err != nil {
			return
		}

		c.JSON(200, gin.H{
			"results":    images,
			"error":      "",
			"total_hits": resultCount,
		})
	})

	r.GET("/api/image/:id", func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			c.JSON(400, gin.H{
				"error": "No ID provided",
			})
			return
		}

		image, err := Database.GetImageEntryFromID(id)
		if err != nil {
			c.JSON(404, gin.H{
				"error": "image not found",
			})
			return
		}

		c.JSON(200, gin.H{
			"image": image,
			"error": "",
		})
	})

	r.GET("/api/image/:id/related", func(c *gin.Context) {
		id := c.Param("id")
		if id == "" {
			c.JSON(400, gin.H{
				"error": "No ID provided",
			})
			return
		}

		ids, err := Database.GetRelatedImageIDs(id)
		if err != nil {
			c.JSON(404, gin.H{
				"error": "image not found",
			})
			return
		}

		c.JSON(200, gin.H{
			"results": ids,
			"error":   "",
		})
	})

	r.Static("/images/", imageDir)

	r.GET("/playground", playgroundHandler())

	gqlGroup := r.Group("")
	gqlGroup.Use(graphqlAuthMiddleware)
	gqlGroup.POST("/query", graphqlHandler())
	gqlGroup.OPTIONS("/query", graphqlHandler())

	r.NoRoute(func(c *gin.Context) {
		c.FileFromFS(c.Request.URL.Path, getFrontendFS())
	})

	err := r.Run()
	if err != nil {
		log.Fatal("Failed to start server:", err)
	} // listen and serve on
}

// Defining the Playground handler
func playgroundHandler() gin.HandlerFunc {
	h := playground.Handler("GraphQL", "/query")

	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

// / Verifies Authorization header to be matching the admin token, if so context contains admin key with true value
func graphqlAuthMiddleware(c *gin.Context) {
	cookie, err := c.Cookie("auth")
	var cookieToken string
	if err != nil {
		cookieToken = cookie
	}

	var headerToken string
	if len(c.Request.Header.Get("Authorization")) > 7 {
		headerToken = c.Request.Header.Get("Authorization")[7:] // remove "Bearer " from token
	}

	if headerToken == "" && cookieToken == "" {
		ctx := context.WithValue(c.Request.Context(), "admin", false)
		c.Request = c.Request.WithContext(ctx)
		return
	}

	if Database.GetAdminToken() != "" && (cookieToken == Database.GetAdminToken() || headerToken == Database.GetAdminToken()) {
		ctx := context.WithValue(c.Request.Context(), "admin", true)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
		return
	}
}

// Defining the Graphql handler
func graphqlHandler() gin.HandlerFunc {
	// NewExecutableSchema and Config are in the generated.go file
	// Resolver is in the resolver.go file
	h := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &graph.Resolver{}}))

	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

func corsMiddleware(c *gin.Context) {
	if Database.GetCorsEnabled() {
		c.Header("Access-Control-Allow-Origin", Database.GetBaseURL())
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		c.Header("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization")
	} else {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, Authorization")
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	}
}
