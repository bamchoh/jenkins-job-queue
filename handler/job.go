package handler

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo"
	"golang.org/x/net/websocket"

	bolt "go.etcd.io/bbolt"

	"test_httpserver/job"
)

type JobHandler struct {
	Db       *bolt.DB
	RootName []byte
	Ch       chan int
	UpdateCh chan int
}

func (jh *JobHandler) MainPage() echo.HandlerFunc {
	return func(c echo.Context) error {
		fp, err := os.Open("./index.html")
		if err != nil {
			return err
		}
		defer fp.Close()
		html, err := ioutil.ReadAll(fp)
		return c.HTML(http.StatusOK, string(html))
	}
}

func (jh *JobHandler) Update() echo.HandlerFunc {
	return func(c echo.Context) error {
		w := c.Response()
		r := c.Request()
		err := job.Update(jh.Db, jh.RootName, w, r)
		close(jh.UpdateCh)
		jh.UpdateCh = make(chan int, 1)
		if err != nil {
			return fmt.Errorf("DB Update Error: %s", err)
		}
		return nil
	}
}

func (jh *JobHandler) WebSocket() echo.HandlerFunc {
	return func(c echo.Context) error {
		websocket.Handler(func(ws *websocket.Conn) {
			defer ws.Close()
			for {
				sMsg := fmt.Sprintln("==============================================")
				sMsg += time.Now().Format("2006-02-03 15:04:05\n")

				jh.Db.View(func(tx *bolt.Tx) error {
					bucket := tx.Bucket(jh.RootName)
					if bucket != nil {
						bucket.ForEach(func(k, v []byte) error {
							sMsg += fmt.Sprintln("==============================================")
							sMsg += fmt.Sprintf("[%s] = \n", string(k))
							idBucket := bucket.Bucket(k)
							if idBucket != nil {
								idBucket.ForEach(func(kk, vv []byte) error {
									if string(kk) == "parameter" {
										sMsg += fmt.Sprintf("  %s = {\n", string(kk))
										paramBucket := idBucket.Bucket(kk)
										paramBucket.ForEach(func(kkk, vvv []byte) error {
											sMsg += fmt.Sprintf("    %s = %s\n",
												string(kkk), string(vvv))
											return nil
										})
										sMsg += fmt.Sprintf("  }\n")
									} else {
										sMsg += fmt.Sprintf("  %s = %s\n",
											string(kk),
											string(vv))
									}
									return nil
								})
							}
							return nil
						})
						return nil
					}
					return nil
				})

				fmt.Println(sMsg)
				err := websocket.Message.Send(ws, sMsg)
				if err != nil {
					c.Logger().Error(err)
					return
				}
				fmt.Println("Waiting...")
				select {
				case <-jh.Ch:
					fmt.Println("Wake Up by executing")
				case <-jh.UpdateCh:
					fmt.Println("Wake Up by updating")
				}
			}
		}).ServeHTTP(c.Response(), c.Request())
		return nil
	}
}
