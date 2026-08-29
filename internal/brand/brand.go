// Package brand is Lamha's application mark: the window logo, themed icon, and tray pixmap.
package brand

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"os"
	"path/filepath"
	"sync"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/gdkpixbuf/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// Name is the freedesktop icon name used by the desktop file and windows.
const Name = "io.github.lamha.Lamha"

// DeveloperName is the public workshop name shown in About Me.
const DeveloperName = "Shenepoy"

// DeveloperProfileURL is the GitHub profile used in About Me.
const DeveloperProfileURL = "https://github.com/Zyzto"

// SourceURL is the public Lamha repository.
const SourceURL = "https://github.com/Zyzto/Lamha"

// UpdateURL is the GitHub Releases page AppImageUpdate and Settings use.
const UpdateURL = SourceURL + "/releases/latest"

// IssuesURL is the public bug tracker.
const IssuesURL = SourceURL + "/issues"

// UpdateInformation is the AppImageUpdate gh-releases-zsync spec.
const UpdateInformation = "gh-releases-zsync|Zyzto|Lamha|latest|Lamha-*x86_64.AppImage.zsync"

// PanelName is a tray-only icon so the panel does not scale the padded SVG to 16px.
const PanelName = Name + "-panel"

//go:embed logo.svg
var SVG []byte

const (
	logoCanvas  = `width="1024" height="1024" viewBox="0 0 1024 1024"`
	panelCanvas = `width="754" height="754" viewBox="135 133 754 754"`
)

var (
	once       sync.Once
	pixmapOnce sync.Once
	themeErr   error
	themeDir   string
	trayDir    string
	pixmaps    []Pixmap
)

// Install writes the logo into the user icon theme and a tray-only search path.
func Install() error {
	once.Do(func() {
		dataHome := os.Getenv("XDG_DATA_HOME")
		if dataHome == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				themeErr = err
				return
			}
			dataHome = filepath.Join(home, ".local", "share")
		}
		themeDir = filepath.Join(dataHome, "icons")
		hicolor := filepath.Join(themeDir, "hicolor", "scalable", "apps")
		if err := os.MkdirAll(hicolor, 0o755); err != nil {
			themeErr = err
			return
		}
		if err := os.WriteFile(filepath.Join(hicolor, Name+".svg"), SVG, 0o644); err != nil {
			themeErr = err
			return
		}
		if err := os.WriteFile(filepath.Join(hicolor, PanelName+".svg"), panelSVG(), 0o644); err != nil {
			themeErr = err
			return
		}

		cache := os.Getenv("XDG_CACHE_HOME")
		if cache == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				themeErr = err
				return
			}
			cache = filepath.Join(home, ".cache")
		}
		trayDir = filepath.Join(cache, "lamha", "icons")
		if err := os.MkdirAll(trayDir, 0o755); err != nil {
			themeErr = err
			return
		}
		if err := os.WriteFile(filepath.Join(trayDir, PanelName+".svg"), panelSVG(), 0o644); err != nil {
			themeErr = err
			return
		}
	})
	return themeErr
}

// ApplyIconTheme makes GTK resolve Name after Install.
func ApplyIconTheme() {
	_ = Install()
	if display := gdk.DisplayGetDefault(); display != nil && themeDir != "" {
		gtk.IconThemeGetForDisplay(display).AddSearchPath(themeDir)
	}
	gtk.WindowSetDefaultIconName(Name)
}

// ThemeDir is the icon-theme root written by Install.
func ThemeDir() string {
	_ = Install()
	return themeDir
}

// TrayThemeDir is a flat directory StatusNotifier hosts can search by IconName.
func TrayThemeDir() string {
	_ = Install()
	return trayDir
}

// File is the installed scalable icon path.
func File() string {
	_ = Install()
	if themeDir == "" {
		return ""
	}
	return filepath.Join(themeDir, "hicolor", "scalable", "apps", Name+".svg")
}

// Pixbuf renders the logo at pixel size.
func Pixbuf(size int) *gdkpixbuf.Pixbuf {
	return renderSVG(size)
}

// TrayPixbuf renders the cropped panel logo so the mark fills a tray slot.
func TrayPixbuf(size int) *gdkpixbuf.Pixbuf {
	if size < 32 {
		size = 32
	}
	if src := renderSVGBytes(panelSVG(), size); src != nil {
		return src
	}
	src := renderSVG(512)
	if src == nil {
		return nil
	}
	if cropped := cropOpaque(src); cropped != nil {
		src = cropped
	}
	if src.Width() == size && src.Height() == size {
		return src
	}
	return src.ScaleSimple(size, size, gdkpixbuf.InterpBilinear)
}

// panelSVG is a copy of the window logo with empty canvas cropped off.
func panelSVG() []byte {
	return bytes.Replace(SVG, []byte(logoCanvas), []byte(panelCanvas), 1)
}

// Image is a GTK image of the logo at pixel size.
func Image(size int) *gtk.Image {
	if pb := Pixbuf(size); pb != nil {
		image := gtk.NewImageFromPixbuf(pb)
		image.SetPixelSize(size)
		image.AddCSSClass("lamha-logo")
		return image
	}
	image := gtk.NewImageFromIconName(Name)
	image.SetPixelSize(size)
	image.AddCSSClass("lamha-logo")
	return image
}

// Pixmap is one StatusNotifierItem ARGB32 frame.
type Pixmap struct {
	Width  int32
	Height int32
	Data   []byte
}

// Pixmaps are tray-sized logo frames. Hosts that ignore IconThemePath still show the mark.
func Pixmaps() []Pixmap {
	_ = Install()
	pixmapOnce.Do(func() {
		for _, size := range []int{128, 96, 64} {
			pb := TrayPixbuf(size)
			if pb == nil {
				continue
			}
			data := argb32(pb)
			if len(data) == 0 {
				continue
			}
			pixmaps = append(pixmaps, Pixmap{Width: int32(pb.Width()), Height: int32(pb.Height()), Data: data})
			if size == 128 && trayDir != "" {
				if png, err := pb.SaveToBufferv("png", nil, nil); err == nil {
					_ = os.WriteFile(filepath.Join(trayDir, PanelName+".png"), png, 0o644)
				}
			}
		}
	})
	return pixmaps
}

func renderSVG(size int) *gdkpixbuf.Pixbuf {
	return renderSVGBytes(SVG, size)
}

func renderSVGBytes(data []byte, size int) *gdkpixbuf.Pixbuf {
	if size < 16 {
		size = 16
	}
	if len(data) == 0 {
		return nil
	}
	loader := gdkpixbuf.NewPixbufLoader()
	loader.SetSize(size, size)
	if err := loader.Write(data); err != nil {
		return nil
	}
	if err := loader.Close(); err != nil {
		return nil
	}
	return loader.Pixbuf()
}

func cropOpaque(pb *gdkpixbuf.Pixbuf) *gdkpixbuf.Pixbuf {
	if pb == nil {
		return nil
	}
	width, height := pb.Width(), pb.Height()
	stride, channels := pb.Rowstride(), pb.NChannels()
	src, alpha := pb.Pixels(), pb.HasAlpha()
	if width < 8 || height < 8 || channels < 3 || len(src) < height*stride {
		return nil
	}
	minX, minY, maxX, maxY := width, height, -1, -1
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			i := y*stride + x*channels
			visible := src[i] > 8 || src[i+1] > 8 || src[i+2] > 8
			if alpha && channels >= 4 {
				visible = src[i+3] > 24
			}
			if !visible {
				continue
			}
			if x < minX {
				minX = x
			}
			if y < minY {
				minY = y
			}
			if x > maxX {
				maxX = x
			}
			if y > maxY {
				maxY = y
			}
		}
	}
	if maxX < minX || maxY < minY {
		return nil
	}
	box := maxX - minX + 1
	if h := maxY - minY + 1; h > box {
		box = h
	}
	pad := box / 24
	if pad < 2 {
		pad = 2
	}
	box += pad * 2
	cx := (minX + maxX) / 2
	cy := (minY + maxY) / 2
	x := cx - box/2
	y := cy - box/2
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x+box > width {
		box = width - x
	}
	if y+box > height {
		box = height - y
	}
	if box < 16 || (box >= width-2 && box >= height-2) {
		return nil
	}
	return pb.NewSubpixbuf(x, y, box, box)
}

func argb32(pb *gdkpixbuf.Pixbuf) []byte {
	if pb == nil {
		return nil
	}
	width, height := pb.Width(), pb.Height()
	stride, channels := pb.Rowstride(), pb.NChannels()
	src, alpha := pb.Pixels(), pb.HasAlpha()
	if width <= 0 || height <= 0 || channels < 3 || len(src) < height*stride {
		return nil
	}
	out := make([]byte, width*height*4)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			i := y*stride + x*channels
			r, g, b, a := src[i], src[i+1], src[i+2], byte(255)
			if alpha && channels >= 4 {
				a = src[i+3]
			}
			binary.BigEndian.PutUint32(out[(y*width+x)*4:], uint32(a)<<24|uint32(r)<<16|uint32(g)<<8|uint32(b))
		}
	}
	return out
}
