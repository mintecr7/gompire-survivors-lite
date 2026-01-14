package game

import (
	"horde-lab/internal/assets"
	"horde-lab/internal/commons/logger_config"

	"github.com/hajimehoshi/ebiten/v2"
)

type AssetState struct {
	Img     *ebiten.Image
	Pending bool
	Err     error
}

type AssetManager struct {
	loader *assets.Loader
	items  map[string]*AssetState
}

func NewAssetManager(loader *assets.Loader) *AssetManager {
	return &AssetManager{
		loader: loader,
		items:  map[string]*AssetState{},
	}
}

// request schedules an asset load if not already loaded/pending.
func (am *AssetManager) Request(key, path string) {
	st := am.items[key]

	if st != nil && (st.Pending || st.Img != nil || st.Err != nil) {
		return
	}

	am.items[key] = &AssetState{Pending: true}

	select {
	case am.loader.Req <- assets.Request{Key: key, Path: path}:
	default:
		// If request queue is full, mark not pending so it can be retried later
		am.items[key].Pending = false
		logger_config.Warnf("[assets] request queue full for key=%s", key)
	}
}

// poll drains loader results and converts decoded images into ebiten.Images
// Call this from Game.Update (main thread).

func (am *AssetManager) Poll() {
	for {
		select {
		case r := <-am.loader.Res:
			st := am.items[r.Key]
			if st == nil {
				st = &AssetState{}
				am.items[r.Key] = st
			}

			st.Pending = false

			if r.Err != nil {
				st.Err = r.Err
				logger_config.Warnf("[assets] load failed key=%s err=%v", r.Key, r.Err)
				continue
			}

			// IMPORTANT: create ebiten.Image on main thread
			st.Img = ebiten.NewImageFromImage(r.Image)

		default:
			return
		}
	}
}

func (am *AssetManager) Get(key string) *ebiten.Image {
	st := am.items[key]
	if st == nil {
		return nil
	}

	return st.Img
}

func (am *AssetManager) Status(key string) (loaded bool, pending bool, err error) {
	st := am.items[key]

	if st == nil {
		return false, false, nil
	}
	return st.Img != nil, st.Pending, st.Err
}
