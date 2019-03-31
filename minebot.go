package main

import (
	"fmt"
	bot "github.com/Tnze/gomcbot"
	"github.com/Tnze/gomcbot/util"
	// auth "github.com/Tnze/gomcbot/authenticate"
	"log"
	"math"
	"time"
)

const scanLen = 160

func main() {
	//Login
	// resp, err := auth.Authenticate("", "")
	// if err != nil {
	// 	panic(err)
	// }
	// Auth := resp.ToAuth()
	Auth := bot.Auth{
		Name: "Tester",
		UUID: "Tester",
		AsTk: "none",
	}
	fmt.Println(Auth)

	//Join server
	g, err := Auth.JoinServer("localhost", 25565)
	if err != nil {
		panic(err)
	}

	//Handle game
	events := g.GetEvents()
	go g.HandleGame()

	for e := range events { //Reciving events
		switch e {
		case bot.PlayerSpawnEvent:
			log.Println("进入游戏")
			go start(g)
		}
	}
}

func start(g *bot.Game) {
	util.CalibratePos(g)
	time.Sleep(time.Second)
	for {
		if loc, ore, ok := searchOre(g); ok {
			log.Println(loc, ore)
			path := findPath(g, loc)
			if path == nil {
				log.Println("找不到路径!")
			} else {
				log.Println(path)
				g.LookAt(float64(loc[0])+0.5, float64(loc[1])+0.5, float64(loc[2])+0.5)

				g.Dig(loc[0], loc[1], loc[2])
				util.CalibratePos(g)
			}

		} else {
			log.Println("找不到钻石")
		}
	}
}

func do(g *bot.Game, path []vec3) {
	for i := len(path) - 1; i >= 0; i-- {
		u := path[i]

		//挖掉上方两个方块
		if !util.NonSolid(g.GetBlock(u[0], u[1]+1, u[2]).String()) {
			g.Dig(u[0], u[1]+1, u[2])
		}
		if !util.NonSolid(g.GetBlock(u[0], u[1]+2, u[2]).String()) {
			g.Dig(u[0], u[1]+2, u[2])
		}
		p := g.GetPlayer()
		x, y, z := p.GetBlockPos()

		if i < len(path)-1 && u[1] > path[i+1][1] {
			//挖掉顶部挡住的方块
			if !util.NonSolid(g.GetBlock(x, y+2, z).String()) {
				g.Dig(x, y+2, z)
			}
			util.TweenJumpTo(g, u[0], u[2])
		} else {
			//挖掉挡住前面的方块
			if (i < len(path)-1 && u[1] < path[i+1][1]) &&
				!util.NonSolid(g.GetBlock(u[0], u[1]+3, u[2]).String()) {
				g.Dig(u[0], u[1]+3, u[2])
			}

			err := util.TweenLineMove(g, float64(u[0])+0.5, float64(u[2])+0.5)
			if err != nil {
				log.Println(err)
				break
			}
		}
		util.CalibratePos(g)
	}
}

func searchOre(g *bot.Game) (loc vec3, ore string, ok bool) {
	log.Println("开始搜索方块")
	p := g.GetPlayer()
	x, y, z := int(math.Floor(p.X)), int(math.Floor(p.Y)), int(math.Floor(p.Z))

	X, Y := 0, 0
	for i := 1; i < scanLen; i++ { //螺旋式向外搜索
		f := i%2*2 - 1
		for j := -i; j < i; j++ {
			if j < 0 {
				X += f
			} else {
				Y += f
			}

			for h := -16; h < 16; h++ {
				if y+h < 0 {
					continue
				}
				if has, ore := checkOre(g, x+X, y+h, z+Y); has {
					if access(g, vec3{x + X, y + h, z + Y}) {
						return vec3{x + X, y + h, z + Y}, ore, true
					} else {
						log.Printf("有个矿在%v但挖不到\n", vec3{x + X, y + h, z + Y})
					}
				}
			}
		}
	}
	return
}

func checkOre(g *bot.Game, x, y, z int) (has bool, ore string) {

	ore = g.GetBlock(x, y, z).String()
	switch ore {
	case "minecraft:diamond_ore":
		has = true
		return
	}

	return false, ""
}

type vec3 [3]int

func findPath(g *bot.Game, destination vec3) []vec3 {
	p := g.GetPlayer()
	player := vec3{int(math.Floor(p.X)), int(math.Floor(p.Y)) - 1, int(math.Floor(p.Z))}
	var open, close chain
	open.insert(node{v: player, G: 0, H: distance(player, destination)})

	for {
		n, ok := open.pop() //从开启列表中选取G+H最小的节点
		if !ok {
			return nil
		}
		close.insert(*n) //将它放入关闭列表中

		list := accessable(g, n.v) //检查所有相邻的节点

		for _, v := range list {
			if v == destination { //如果已经到达终点
				path := []vec3{v}
				for {
					path = append(path, n.v)
					if n = n.p; n == nil {
						return path
					}
				}
			} else if nx, ok := open.find(v); ok { //如果已经存在于开启列表了，检查是否路径更短
				w := 1.0
				if !util.NonSolid(g.GetBlock(v[0], v[1]+1, v[2]).String()) {
					w++
				}
				if !util.NonSolid(g.GetBlock(v[0], v[1]+2, v[2]).String()) {
					w++
				}
				if n.G+w < nx.G {
					nx.p, nx.G = n, n.G+w
				}
			} else if _, ok := close.find(v); !ok { //否则如果不在关闭列表，则将它加入开启列表

				open.insert(node{v: v, G: n.G + 1, H: distance(v, destination), p: n})

			}
		}
	}
}

var dirs = [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

//列出附近可到达的位置列表
func accessable(g *bot.Game, pos vec3) (list []vec3) {
	for i := -1; i < 2; i++ {
		if pos[1]+i < 0 {
			continue
		}
		for _, dir := range dirs {
			loc := vec3{pos[0] + dir[0], pos[1] + i, pos[2] + dir[1]}
			bs := g.GetBlock(loc[0], loc[1], loc[2]).String()
			if soild(bs) && access(g, loc) {
				list = append(list, loc)
			}
		}
	}
	return
}

//判断一个位置是否可到达
func access(g *bot.Game, loc vec3) bool {
	h, f :=
		g.GetBlock(loc[0], loc[1]+2, loc[2]).String(),
		g.GetBlock(loc[0], loc[1]+1, loc[2]).String()

	//如果h不可通过并且不能挖掘
	if (!util.NonSolid(h) &&
		!digable(g, vec3{loc[0], loc[1] + 2, loc[2]})) ||
		//或者挖掘后上方有受重力影响的方块
		gravity(g.GetBlock(loc[0], loc[1]+3, loc[2]).String()) {
		return false
	}

	//如果f不可通过且不能挖掘
	if !util.NonSolid(f) && !digable(g, vec3{loc[0], loc[1] + 1, loc[2]}) {
		return false
	}

	return true
}

//是否可以踩踏
func soild(bs string) bool {
	return bs == "minecraft:stone" ||
		bs == "minecraft:diorite" /*闪长岩*/ ||
		bs == "minecraft:granite" /*花岗岩*/ ||
		bs == "minecraft:andesite" /*安山岩*/ ||
		bs == "minecraft:grass_block" ||
		bs == "minecraft:dirt" ||
		bs == "minecraft:glass" ||
		bs == "minecraft:send" ||
		bs == "minecraft:bedrock" ||
		bs == "minecraft:diamond_ore" //钻石矿本身
}

var neighbours = [][3]int{
	{1, 0, 0}, {-1, 0, 0},
	{0, 1, 0}, {0, -1, 0},
	{0, 0, 1}, {0, 0, -1},
}

//是否可以挖掘
func digable(g *bot.Game, loc vec3) bool {
	bs := g.GetBlock(loc[0], loc[1], loc[2]).String()

	if liquid(bs) {
		return false
	}

	if bs != "minecraft:stone" /*石头*/ &&
		bs != "minecraft:diorite" /*闪长岩*/ &&
		bs != "minecraft:granite" /*花岗岩*/ &&
		bs != "minecraft:andesite" /*安山岩*/ &&
		bs != "minecraft:diamond_ore" &&
		bs != "minecraft:dirt" {
		return false
	}

	for _, near := range neighbours { //检查相邻方块是否是液体
		nbh := g.GetBlock(loc[0]+near[0], loc[1]+near[1], loc[2]+near[2]).String()
		if liquid(nbh) {
			return false
		}
	}

	return true
}

//是否是液体
func liquid(bs string) bool {
	return bs == "minecraft:water" ||
		bs == "minecraft:lava"
}

func gravity(bs string) bool {
	return bs == "minecraft:sand" ||
		bs == "minecraft:red_sand" ||
		bs == "minecraft:gravel"
	//忽略混凝土粉末
}

func distance(l, r vec3) float64 {
	return math.Abs(float64(l[0]-r[0])) +
		math.Abs(float64(l[1]-r[1])) +
		math.Abs(float64(l[2]-r[2]))
}

type chain struct {
	head *node
}

type node struct {
	v    vec3
	G, H float64
	p    *node
	next *node
}

func (n *node) lessThan(m *node) bool {
	return n.G+n.H < m.G+m.H
}

func (c *chain) insert(v node) {
	if c.head == nil {
		c.head = &v
		return
	}

	point := c.head

	for point.next != nil && point.next.lessThan(&v) {
		point = point.next
	}
	point.next, v.next = &v, point.next
}

func (c *chain) pop() (n *node, ok bool) {
	if c.head == nil {
		return nil, false
	}
	ok = true

	n = c.head
	c.head = c.head.next
	return
}

func (c *chain) find(v vec3) (n *node, ok bool) {
	pointer := c.head
	for pointer != nil {
		if pointer.v == v {
			return pointer, true
		}
		pointer = pointer.next
	}
	return nil, false
}
