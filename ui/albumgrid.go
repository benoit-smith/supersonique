package ui

import (
	"context"
	"supersonic/backend"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
	"github.com/dweymouth/go-subsonic"
)

type AlbumGrid struct {
	widget.BaseWidget

	grid         *widget.GridWrapList
	albums       []*subsonic.AlbumID3
	iter         backend.AlbumIterator
	imageManager *backend.ImageManager
	fetching     bool
	done         bool
}

var _ fyne.Widget = (*AlbumGrid)(nil)

func NewAlbumGrid(iter backend.AlbumIterator, pm *backend.PlaybackManager, im *backend.ImageManager) *AlbumGrid {
	ag := &AlbumGrid{
		iter:         iter,
		imageManager: im,
	}
	ag.ExtendBaseWidget(ag)

	g := widget.NewGridWrapList(
		func() int {
			return len(ag.albums)
		},
		// create func
		func() fyne.CanvasObject {
			ac := NewAlbumCard()
			ac.OnTapped = func() {
				pm.PlayAlbum(ac.AlbumID())
			}
			return ac
		},
		// update func
		func(itemID int, obj fyne.CanvasObject) {
			ac := obj.(*AlbumCard)
			ag.doUpdateAlbumCard(itemID, ac)
		},
	)
	ag.grid = g

	// fetch initial albums
	ag.fetchMoreAlbums(36)
	return ag
}

func (ag *AlbumGrid) doUpdateAlbumCard(albumIdx int, ac *AlbumCard) {
	album := ag.albums[albumIdx]
	if ac.PrevAlbumID == album.ID {
		// nothing to do
		return
	}
	ac.Update(album)
	ac.PrevAlbumID = album.ID
	// TODO: set image to a placeholder before spinning off async fetch
	// cancel any previous image fetch
	if ac.ImgLoadCancel != nil {
		ac.ImgLoadCancel()
		ac.ImgLoadCancel = nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func(ctx context.Context) {
		i, err := ag.imageManager.GetAlbumThumbnail(album.ID)
		select {
		case <-ctx.Done():
			return
		default:
			if err == nil {
				ac.Cover.SetImage(i)
				ac.Refresh()
			}
		}
	}(ctx)
	ac.ImgLoadCancel = cancel

	// TODO: remove magic number 10
	if !ag.done && !ag.fetching && albumIdx > len(ag.albums)-10 {
		ag.fetchMoreAlbums(10)
	}
}

func (a *AlbumGrid) fetchMoreAlbums(count int) {
	if a.iter == nil {
		a.done = true
	}
	i := 0
	a.fetching = true
	a.iter.NextN(count, func(al *subsonic.AlbumID3) {
		if al == nil {
			a.done = true
			return
		}
		a.albums = append(a.albums, al)
		i++
		if i == count {
			a.fetching = false
		}
	})
}

func (a *AlbumGrid) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(a.grid)
}
