package handlers

import (
	"time"

	"qingye/server/models"
	"qingye/server/services"

	"github.com/gin-gonic/gin"
)

// plantBody 植物创建/更新请求体（acquiredDate 支持 YYYY-MM-DD 或 RFC3339）
type plantBody struct {
	Name         string `json:"name"`
	Species      string `json:"species"`
	Photo        string `json:"photo"`
	RoomID       uint   `json:"roomId"`
	Note         string `json:"note"`
	Location     string `json:"location"`
	LightReq     string `json:"lightReq"`
	AcquiredDate string `json:"acquiredDate"`
	Attributes   string `json:"attributes"`
}

// toModel 解析请求体为植物模型（处理日期字符串）
func (b *plantBody) toModel(id uint) (*models.Plant, error) {
	p := &models.Plant{
		ID:         id,
		Name:       b.Name,
		Species:    b.Species,
		Photo:      b.Photo,
		RoomID:     b.RoomID,
		Note:       b.Note,
		Location:   b.Location,
		LightReq:   b.LightReq,
		Attributes: b.Attributes,
	}
	if b.AcquiredDate != "" {
		var t time.Time
		var err error
		if t, err = time.Parse("2006-01-02", b.AcquiredDate); err != nil {
			if t, err = time.Parse(time.RFC3339, b.AcquiredDate); err != nil {
				return nil, err
			}
		}
		p.AcquiredDate = &t
	}
	return p, nil
}

// PlantHandler 植物与房间
type PlantHandler struct {
	plants *services.PlantService
	rooms  *services.RoomService
}

func NewPlantHandler() *PlantHandler {
	return &PlantHandler{
		plants: services.NewPlantService(),
		rooms:  services.NewRoomService(),
	}
}

// ---- 房间 ----

func (h *PlantHandler) ListRooms(c *gin.Context) {
	rooms, err := h.rooms.ListWithStats()
	if err != nil {
		ServerError(c, "获取房间失败")
		return
	}
	OK(c, rooms)
}

type roomBody struct {
	Name      string `json:"name"`
	Sort      int    `json:"sort"`
	IsOutdoor bool   `json:"isOutdoor"`
	Icon      string `json:"icon"`
}

func (h *PlantHandler) CreateRoom(c *gin.Context) {
	var body roomBody
	if err := c.ShouldBindJSON(&body); err != nil {
		BadRequest(c, "请求格式错误")
		return
	}
	room, err := h.rooms.Create(&models.Room{Name: body.Name, Sort: body.Sort, IsOutdoor: body.IsOutdoor, Icon: body.Icon})
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, room)
}

func (h *PlantHandler) UpdateRoom(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var body roomBody
	if err := c.ShouldBindJSON(&body); err != nil {
		BadRequest(c, "请求格式错误")
		return
	}
	room, err := h.rooms.Update(&models.Room{ID: id, Name: body.Name, Sort: body.Sort, IsOutdoor: body.IsOutdoor, Icon: body.Icon})
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, room)
}

func (h *PlantHandler) DeleteRoom(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.rooms.Delete(id); err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, nil)
}

// ---- 植物 ----

func (h *PlantHandler) ListPlants(c *gin.Context) {
	roomID, _ := parseUintQuery(c, "roomId")
	list, err := h.plants.List(roomID)
	if err != nil {
		ServerError(c, "获取植物列表失败")
		return
	}
	OK(c, list)
}

func (h *PlantHandler) GetPlant(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	p, err := h.plants.Get(id)
	if err != nil {
		NotFound(c, "植物不存在")
		return
	}
	OK(c, p)
}

func (h *PlantHandler) CreatePlant(c *gin.Context) {
	var body plantBody
	if err := c.ShouldBindJSON(&body); err != nil {
		BadRequest(c, "请求格式错误")
		return
	}
	model, err := body.toModel(0)
	if err != nil {
		BadRequest(c, "acquiredDate 格式错误，应为 YYYY-MM-DD")
		return
	}
	p, err := h.plants.Create(model)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, p)
}

func (h *PlantHandler) UpdatePlant(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var body plantBody
	if err := c.ShouldBindJSON(&body); err != nil {
		BadRequest(c, "请求格式错误")
		return
	}
	model, err := body.toModel(id)
	if err != nil {
		BadRequest(c, "acquiredDate 格式错误，应为 YYYY-MM-DD")
		return
	}
	p, err := h.plants.Update(model)
	if err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, p)
}

func (h *PlantHandler) DeletePlant(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.plants.Delete(id); err != nil {
		BadRequest(c, err.Error())
		return
	}
	OK(c, nil)
}
