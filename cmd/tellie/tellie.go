 package main

import (
	"errors"
	"math"
	"os"
	"regexp"
	"fmt"
	"strings"
	"strconv"
	"log"
	"time"

	"github.com/beefsack/go-astar"
	"github.com/google/uuid"

	"github.com/Tnze/go-mc/yggdrasil"
	"github.com/Tnze/go-mc/bot"
	"github.com/Tnze/go-mc/bot/path"
	// "github.com/Tnze/go-mc/bot/world/entity/player"
	"github.com/Tnze/go-mc/chat"
	"github.com/Tnze/go-mc/realms"
	_ "github.com/Tnze/go-mc/data/lang/en-us"
	"github.com/ugjka/cleverbot-go"
)

const timeout = 45
const username string = "lanelawley@gmail.com"
const password string = "MbuRobots2"
const realm_name string = "butt"

var (
	r     *realms.Realms
	c     *bot.Client
	realm_address = ""
	realm_port = 0

	warping = false
	xbase,ybase,zbase int
	ship_xl,ship_yl,ship_zl,ship_xu,ship_yu,ship_zu int
	ship_x,ship_y,ship_z int
	locations = make(map[string][3]int)

	watch chan time.Time
	apiKey = "CC238ZlLq4J0m-JTvrKBlmx5XNA"
	re = regexp.MustCompile("[A-Z]+:")
	re2 = regexp.MustCompile("\\.\\!\\?")

	curPath []astar.Pather

	playerUUIDToName = map[string]string{
		"cc4bd981-aa2f-4e9e-9a62-30f520115f27": "CowSnail",
		"abacf297-e722-445e-ad4b-484221445875": "scefing",
	}
)


var session = cleverbot.New(apiKey)


func main() {
	c = bot.NewClient()
	xbase = 90
	ybase = 66
	zbase = -247

	// log in
	auth, err := yggdrasil.Authenticate(username,password)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	c.Auth.UUID, c.Name = auth.SelectedProfile()
	c.AsTk = auth.AccessToken()

	fmt.Println("user:", c.Name)
	fmt.Println("uuid:", c.Auth.UUID)
	fmt.Println("astk:", c.AsTk)

	// parse realms
	r = realms.New("1.16.3", c.Name, c.AsTk, c.Auth.UUID)
	servers,err := r.Worlds()

	if err != nil {
		panic(err)
	}

	for _,v := range servers {
		if v.Name == realm_name {
			fmt.Println("Found Realm", realm_name)
			fmt.Printf("v is %s\n", v)
			address, err := r.Address(v)
			if err != nil {
				panic(err)
			}
			rholder := strings.SplitN(address,":",2)
			realm_address = rholder[0]
			realm_port,err = strconv.Atoi(rholder[1])
			fmt.Println(realm_address, realm_port)
		}
	}
	if realm_address == "" {
		panic("Realm not found!")
	}

	// join server
	if err := c.JoinServer(realm_address, realm_port); err != nil {
		log.Fatal(err)
	}
	log.Println("Login success")

	//Register event handlers
	c.Events.GameStart = onGameStart
	c.Events.ChatMsg = onChatMsg
	c.Events.Disconnect = onDisconnect
	c.Events.SoundPlay = onSound
	c.Events.Die = onDeath
	c.Events.PrePhysics = onPhys

	//JoinGame
	err = c.HandleGame()
	if err != nil {
		log.Fatal(err)
	}
}
func onDeath() error {
	log.Println("Death")

	c.Chat("Respawning...")
	c.Respawn()

	if warping == false {
		c.Chat(fmt.Sprintf("/teleport Telleilogical %d %d %d",xbase,ybase,zbase))
	}
	return nil
}

func onGameStart() error {
	log.Println("Game start")

	c.Chat("hello")

	locations["skyhold"] = [3]int{5500,200,4500}
	locations["base"] = [3]int{130,70,-240}
/*
	locations["skyhold"][0] = 5500
	locations["skyhold"][1] = 200
	locations["skyhold"][2] = 4500

	locations["base"][0] = 130
	locations["base"][1] = 70
	locations["base"][2] = -240
*/
	watch = make(chan time.Time)
	return nil
}

func onSound(name string, category int, x, y, z float64, volume, pitch float32) error {
	return nil
}

func leave() int {
	// Sign out
	err := yggdrasil.SignOut(username, password)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	os.Exit(0)
	return 0
}


func Max(x, y int) int {
	if x > y {
		return x
	} else {
		return y
	}
}
func Min(x, y int) int {
	if x < y {
		return x
	} else {
		return y
	}
}


func bed(xb,yb,zb int) error {
	log.Println("Bed requested")
	// look for a bed nearby
	err := c.UseBlock(0,xb,yb,zb,1,0.5,1,0.5,false)
	if err != nil {
		return err
	}
	err = c.UseBlock(0,xb,yb,zb,1,0.5,1,0.5,false)
	if err != nil {
		return err
	}
	log.Println("Done")
	return nil
}

/*var pos player.Pos
var nav *path.Nav
var thePath []astar.Pather
var found bool
var tileStarted time.Time*/

type pathRet struct{
	path []astar.Pather
	err error
}

func doPath(c1 chan pathRet, x, y, z int) {
	pos := c.Player.Pos

	nav := &path.Nav{
		World: &c.Wd,
		Start: path.V3{X: int(math.Floor(pos.X)), Y: int(math.Floor(pos.Y - 0.6)), Z: int(math.Floor(pos.Z))},
		// Dest: path.V3{X: int(math.Floor(pos.X)) + 1, Y: int(math.Floor(pos.Y - 0.6)), Z: int(math.Floor(pos.Z))},
		Dest: path.V3{X: x, Y: y, Z: z},
	}

	path, _, found := nav.Path()
	if !found {
		// return nil, errors.New("no path")
		c1 <- pathRet{
			path: nil,
			err: errors.New("no path"),
		}
	} else {
		c1 <- pathRet{
			path: path,
			err: nil,
		}
	}
}

func onPhys() error {
	if len(curPath) <= 0 {
		c.Inputs = path.Inputs{}
		return nil
	}

	pos := c.Player.Pos

	start:
	next := curPath[len(curPath)-1].(path.Tile)

	dx, dy, dz := pos.X-float64(next.Pos.X)-0.48, pos.Y-float64(next.Pos.Y)-1, pos.Z-float64(next.Pos.Z)-0.48
	if next.IsComplete(path.Point{X: dx, Y: dy, Z: dz}) {
			fmt.Printf("next path marker is %s\n", next.Pos)
			curPath = curPath[:len(curPath)-1]
			if len(curPath) > 0 {
				goto start
			}
	}

	inputs := next.Inputs(
		path.Point{X: pos.X, Y: pos.Y, Z: pos.Z},
		path.Point{X: dx, Y: dy, Z: dz},
		path.Point{X: 1, Y: 1, Z: 1},
		20 * time.Millisecond,
	)

	c.Inputs = inputs

	// fmt.Printf("%s\n", inputs)
	// c.Chat(fmt.Sprintf("/teleport Telleilogical %d %d %d", next.Pos.X, next.Pos.Y, next.Pos.Z))

	return nil
}

func onChatMsg(cm chat.Message, pos byte, uuid uuid.UUID) error {
	cmstr := cm.String()
	log.Println("Chat:", cmstr)

	// cmstr := cm.String()
	if false == true {
		// this is just here for now.
	} else {
		// it's a standard message.
		var spl []string
		// var spl, spl2 []string

		if len(cmstr) == 0 {
			log.Println("empty chat message")
			return nil
		}

		if cmstr[0] == '[' {
			spl = strings.Split(cmstr, "] ")
			// spl2 = strings.Split(spl[0],"[")
		} else if cmstr[0] == '<' {
			spl = strings.Split(cmstr, "> ")
			// spl2 = strings.Split(spl[0],"<")
		} else {
			// return nil
			spl = []string{"", cmstr}
		}
		if len(spl) <= 1 {
			return nil
		}

		msg := spl[1]
		// requester := spl2[1]
		if len(msg) > 2 && strings.ToLower(msg[:3]) == "bed" {
			err := bed(-8576,69,-1995)
			if err != nil {
				log.Fatal(err)
			}
		} else if len(msg) > 3 && strings.ToLower(msg[:4]) == "come" {
			requester := spl[0][1:]
			players := c.Wd.PlayerEntities()
			for _, ent := range players {
				// c.Chat(fmt.Sprintf("player found: %+v\n", ent))
				if playerUUIDToName[ent.UUID.String()] == requester {
					c1 := make(chan pathRet, 1)
					x := int(math.Floor(ent.X))
					y := int(math.Floor(ent.Y - 0.6))
					z := int(math.Floor(ent.Z))
					go doPath(c1, x, y, z)
					c.Chat(fmt.Sprintf("going to %s at %d, %d, %d", requester, x, y, z))
					select {
					case pr := <-c1:
						path := pr.path
						err := pr.err
						if err != nil {
							c.Chat("No path found")
						} else {
								c.Chat(fmt.Sprintf("Found a path (length %d)", len(path)))
								fmt.Printf("%s\n", path)
								for p, i := range path {
									fmt.Printf("path %d: %s\n", i, p)
								}
								curPath = path
						}
					case <-time.After(5 * time.Second):
						c.Chat("No path found (timed out searching)")
					}
				}
			}
			// c.Chat(fmt.Sprintf("want to go to %s at %d %d %d\n", requester, -1, -1, -1))
		} else if len(msg) > 3 && strings.ToLower(msg[:4]) == "walk" {
			coords := strings.Split(msg, " ")[1:]
			x, _ := strconv.Atoi(coords[0])
			y, _ := strconv.Atoi(coords[1])
			z, _ := strconv.Atoi(coords[2])

			c1 := make(chan pathRet, 1)
			go doPath(c1, x, y, z)
			select {
			case pr := <-c1:
				path := pr.path
				err := pr.err
				if err != nil {
					c.Chat("No path found")
				} else {
						c.Chat(fmt.Sprintf("Found a path (length %d)", len(path)))
						fmt.Printf("%s\n", path)
						for p, i := range path {
							fmt.Printf("path %d: %s\n", i, p)
						}
						curPath = path
				}
			case <-time.After(5 * time.Second):
				c.Chat("No path found (timed out searching)")
			}

		} else if msg == "You can only sleep at night and during thunderstorms" {
			c.Chat("I can't sleep just yet.")
		} else if msg == "This bed is occupied" {
			c.Chat("In bed")
		} else if len(msg) > 6 && strings.ToLower(msg[:6]) == "tellie" {
			mspl := strings.Split(msg, " ")
			pmsg := msg
			if len(mspl) > 1 {
				pmsg = strings.Join(mspl[1:], " ")
			}

			if pmsg == "leave" {
				log.Println("Requested to leave")
				leave()
			} else if len(pmsg) > 6 && strings.ToLower(pmsg[:6]) == "select" {
				j, err := strconv.Atoi(mspl[2])
				if err != nil {
					c.Chat("I don't understand that slot.")
					return nil
				} else if j > 8 || j < 0 {
					c.Chat("That slot isn't valid.")
				}
				c.SelectItem(j)
			} else if pmsg == "what are you holding" {
				c.Chat(fmt.Sprintf("%d", c.Player.HeldItem))
			} else {
				resp, err := session.Ask(pmsg)
				if err != nil {
					fmt.Printf("Cleverbot error: %v\n", err)
				} else {
					c.Chat(resp)
				}
				/*
				inp := fmt.Sprintf("MAN: %s WOMAN: ", pmsg)
				out, err := exec.Command("/bin/bash", "./cmd.sh", inp).Output()
				if err != nil {
					// log.Fatal(err)
					fmt.Printf("GPT2 error: %v\n", err)
				}
				proc := re.Split(string(out), -1)
				tellieResp := strings.Split(strings.Trim(proc[2], " 	\n"), "\n")[0]
				proc2 := re2.Split(tellieResp, -1)
				if len(proc2) > 1 {
					tellieResp = strings.Join(proc2[:len(proc2)-1], " ")
				}
				c.Chat(tellieResp)
				*/
			}
		}
	}

	return nil
}

func onDisconnect(c chat.Message) error {
	log.Println("Disconnect:", c)
	return nil
}

