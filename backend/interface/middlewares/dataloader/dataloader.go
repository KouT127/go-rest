package dataloader

import (
	"context"
	"fmt"
	"github.com/KouT127/gin-sample/backend/domain/model"
	"github.com/KouT127/gin-sample/backend/infrastracture/database"
	"github.com/gin-gonic/gin"
	"strconv"
	"strings"
	"time"
)

type ctxKeyType struct{ name string }

var ctxKey = ctxKeyType{"appCtx"}

type Loaders struct {
	UserById   *UserLoader
	TaskByUser *TaskSliceLoader
}

func LoaderMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ldrs := Loaders{}
		wait := 250 * time.Microsecond
		ldrs.UserById = &UserLoader{
			wait:     wait,
			maxBatch: 100,
			fetch: func(keys []int) (users []*model.User, errors []error) {
				var keySql []string
				idx := 0
				for _, key := range keys {
					keySql = append(keySql, strconv.Itoa(key))
				}
				errors = make([]error, len(keys))
				db := database.NewDB()
				time.Sleep(5 * time.Millisecond)
				query := db.Table("users").Where("id in (?)", strings.Join(keySql, ","))
				rows, err := query.Rows()
				if err != nil {
					users = append(users, &model.User{})
					errors = append(errors, err)
					return users, errors
				}
				users = make([]*model.User, len(keys))
				for i, _ := range keys {
					rows.Next()
					u := &model.User{}
					err := db.ScanRows(rows, u)
					if err != nil {
						errors = append(errors, err)
					}
					users[i] = u
					idx += 1
				}
				return users, errors
			},
		}
		ldrs.TaskByUser = &TaskSliceLoader{
			wait:     wait,
			maxBatch: 100,
			fetch: func(keys []int) (tasks [][]*model.Task, errors []error) {
				var keySql []string
				for _, key := range keys {
					keySql = append(keySql, strconv.Itoa(key))
				}
				errors = make([]error, len(keys))
				tasks = make([][]*model.Task, len(keys))
				db := database.NewDB()
				time.Sleep(5 * time.Millisecond)

				var ts []model.Task
				query := db.Table("tasks").Where("user_refer in (?)", strings.Join(keySql, ",")).Scan(&ts)
				_, err := query.Rows()
				if err != nil {
					tasks = append(tasks, []*model.Task{})
					errors = append(errors, err)
					return tasks, errors
				}
				for i, key := range keys {
					for _, task := range ts {
						if int(task.UserRefer) == key {
							tasks[i] = append(tasks[i], &task)
						}
					}
				}
				return tasks, errors
			},
		}
		ctx := context.WithValue(c.Request.Context(), ctxKey, ldrs)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func CtxLoaders(ctx context.Context) (Loaders, error) {
	gCtx := ctx.Value(ctxKey)
	if gCtx == nil {
		err := fmt.Errorf("could not retrieve gin.Context")
		return Loaders{}, err
	}
	ldr := gCtx.(Loaders)
	return ldr, nil
}
