package aoi_list

import (
	"errors"
	"fmt"
	"github.com/go-redis/redis"
	"go-aoi/redisDB"
	"sync"
)

type Scene struct {
	SceneName string
	XListKey  string
	YListKey  string
	Mp        sync.Map //保存本场景的所有对象
}

func (s *Scene) Destory() {
	s.Mp = sync.Map{}
	redisDB.Redis.Del(s.XListKey, s.YListKey)
}

func Create(sceneName, xListKey, yListKey string) *Scene {
	s := new(Scene)
	s.SceneName = sceneName
	s.Mp = sync.Map{}
	s.XListKey = xListKey
	s.YListKey = yListKey
	return s
}

// 返回当前实体进入时的邻居列表
func (s *Scene) Add(ent *Entity) ([]string, error) {
	_, tf := s.Mp.Load(ent.UUID)
	if tf {
		return nil, errors.New(`existed`)
	}
	s.Mp.Store(ent.UUID, ent)
	err := s.handleInsert(ent)
	if err != nil {
		return nil, errors.New(fmt.Sprintf(`scene_handleInsert_err:%s`, err.Error()))
	}
	arr, err := s.handleGetList(ent)
	if err != nil {
		return nil, errors.New(fmt.Sprintf(`scene_handleGetList_err:%s`, err.Error()))
	}
	return arr, nil
}

func (s *Scene) handleGetList(ent *Entity) ([]string, error) {
	data, err := s.handleGetData(ent)
	if err != nil {
		return nil, err
	}
	arr := make([]string, 0, 100)
	for k := range data.Min {
		_, tf := data.Max[k]
		if tf {
			arr = append(arr, k)
		}
	}
	return arr, nil
}

func (s *Scene) handleGetMap(ent *Entity) (map[string]struct{}, error) {
	data, err := s.handleGetData(ent)
	if err != nil {
		return nil, err
	}
	mp := make(map[string]struct{})
	for k := range data.Min {
		_, tf := data.Max[k]
		if tf {
			mp[k] = struct{}{}
		}
	}
	return mp, nil
}

type xyData struct {
	Min map[string]struct{}
	Max map[string]struct{}
}

func (s *Scene) handleGetData(ent *Entity) (*xyData, error) {
	wg := sync.WaitGroup{}
	wg.Add(2)
	r := int(ent.Radius)
	var mpX, mpY map[string]struct{}
	var err error
	go func(wgp *sync.WaitGroup) {
		defer wg.Done()
		min := ent.X - r
		max := ent.X + r
		mpX, err = getList(s.XListKey, min, max)
	}(&wg)
	go func(wgp *sync.WaitGroup) {
		defer wg.Done()
		min := ent.Y - r
		max := ent.Y + r
		mpY, err = getList(s.YListKey, min, max)
	}(&wg)
	wg.Wait()
	if err != nil {
		return nil, err
	}
	delete(mpX, ent.UUID) //删除实体自身
	delete(mpY, ent.UUID) //删除实体自身
	xLen := len(mpX)
	yLen := len(mpY)
	min := mpX
	max := mpY
	if xLen > yLen { //减少遍历次数
		min = mpY
		max = mpX
	}
	data := new(xyData)
	data.Min = min
	data.Max = max
	return data, err
}

func (s *Scene) handleInsert(ent *Entity) error {
	var err error
	var wg sync.WaitGroup
	wg.Add(2)
	go func(wgp *sync.WaitGroup) {
		defer wg.Done()
		err = insert(s.XListKey, ent.UUID, float64(ent.X))
	}(&wg)
	go func(wgp *sync.WaitGroup) {
		defer wg.Done()
		err = insert(s.YListKey, ent.UUID, float64(ent.Y))
	}(&wg)
	wg.Wait()
	if err != nil {
		return err
	}
	return nil
}

func insert(listKey, uuid string, score float64) error {
	_, err := redisDB.Redis.ZAdd(listKey, redis.Z{score, uuid}).Result()
	return err
}

func update(listKey, uuid string, score float64) error {
	_, err := redisDB.Redis.ZAddXX(listKey, redis.Z{score, uuid}).Result()
	return err
}

// 获取指定对象的AOI对象列表
func getList(key string, min, max int) (map[string]struct{}, error) {
	opt := redis.ZRangeBy{
		Min:    fmt.Sprintf(`%d`, min),
		Max:    fmt.Sprintf(`%d`, max),
		Offset: 0,
		Count:  -1,
	}
	mp := map[string]struct{}{}
	arr, err := redisDB.Redis.ZRangeByScore(key, opt).Result()
	if err != nil {
		return mp, err
	}
	for _, val := range arr {
		mp[val] = struct{}{}
	}
	return mp, nil
}

// 更新实体坐标
func (s *Scene) updateEntityPosition(ent *Entity, x, y int) error {
	var err error
	var wg sync.WaitGroup
	wg.Add(2)
	go func(wgp *sync.WaitGroup) {
		defer wg.Done()
		err = update(s.XListKey, ent.UUID, float64(x))
	}(&wg)
	go func(wgp *sync.WaitGroup) {
		defer wg.Done()
		err = update(s.YListKey, ent.UUID, float64(y))
	}(&wg)
	wg.Wait()
	if err != nil {
		return err
	}
	ent.X = x
	ent.Y = y
	return nil
}

func (s *Scene) remove(uuid string) error {
	var err error
	var wg sync.WaitGroup
	wg.Add(2)
	go func(wgp *sync.WaitGroup) {
		defer wg.Done()
		err = remove(s.XListKey, uuid)
	}(&wg)
	go func(wgp *sync.WaitGroup) {
		defer wg.Done()
		err = remove(s.YListKey, uuid)
	}(&wg)
	wg.Wait()
	if err != nil {
		return err
	}
	return nil
}

func remove(listKey, uuid string) error {
	_, err := redisDB.Redis.ZRem(listKey, uuid).Result()
	return err
}

// 返回的是当前实体离开时,要通知的邻居列表
func (s *Scene) Leave(ent *Entity) ([]string, error) {
	_, tf := s.Mp.Load(ent.UUID)
	if !tf {
		return nil, errors.New(`not exist`)
	}
	arr, err := s.handleGetList(ent)
	if err != nil {
		return nil, err
	}
	err = s.remove(ent.UUID)
	if err != nil {
		return nil, err
	}
	s.Mp.Delete(ent.UUID)
	return arr, nil
}

type GetList struct {
	Move  []string //当前实体移动时,要通知的邻居列表
	Add   []string //当前实体进入时,要通知的邻居列表
	Leave []string //当前实体离开时,要通知的邻居列表
}

func (s *Scene) Move(ent *Entity, x, y int) (*GetList, error) {
	_, tf := s.Mp.Load(ent.UUID)
	if !tf {
		return nil, errors.New(`not exist`)
	}
	oldList, err := s.handleGetMap(ent)
	if err != nil {
		return nil, errors.New(fmt.Sprintf(`before-move_handleGetList_err:%s`, err.Error()))
	}
	err = s.updateEntityPosition(ent, x, y)
	if err != nil {
		return nil, errors.New(fmt.Sprintf(`scene_move_updateEntityPosition_err:%s`, err.Error()))
	}
	newList, err := s.handleGetMap(ent)
	if err != nil {
		return nil, errors.New(fmt.Sprintf(`after-move_handleGetList_err:%s`, err.Error()))
	}
	num := (len(oldList) + len(newList)) / 2
	data := new(GetList)
	data.Move = make([]string, 0, num)
	data.Add = make([]string, 0, num)
	data.Leave = make([]string, 0, num)
	for k := range oldList {
		_, existed := newList[k]
		if existed {
			data.Move = append(data.Move, k) //交集
		} else {
			data.Leave = append(data.Leave, k) //在新表中不存在,已离开
		}
	}
	for _, v := range data.Move {
		delete(newList, v) //删除交集,则剩下是新增的
	}
	for k := range newList {
		data.Add = append(data.Add, k)
	}
	return data, nil
}
