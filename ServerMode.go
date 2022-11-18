package main

import (
	"Paktum/Database"
	"Paktum/graph"
	"Paktum/graph/generated"
	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"strconv"
	"time"
)

func ServerMode(imageDir string) {

	rand.Seed(time.Now().UnixNano())
	r := gin.Default()

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

	r.POST("/query", graphqlHandler())

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

// Defining the Graphql handler
func graphqlHandler() gin.HandlerFunc {
	// NewExecutableSchema and Config are in the generated.go file
	// Resolver is in the resolver.go file
	h := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &graph.Resolver{}}))

	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}
