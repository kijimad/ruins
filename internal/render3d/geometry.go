package render3d

import "math"

// Vec は3D空間の点またはベクトル。Y が高さで、XZ がタイル平面に対応する
type Vec struct{ X, Y, Z float64 }

// Add は2つのベクトルを足す
func Add(a, b Vec) Vec { return Vec{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }

// Scale はベクトルを定数倍する
func Scale(a Vec, s float64) Vec { return Vec{a.X * s, a.Y * s, a.Z * s} }

func sub(a, b Vec) Vec     { return Vec{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func dot(a, b Vec) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

func cross(a, b Vec) Vec {
	return Vec{a.Y*b.Z - a.Z*b.Y, a.Z*b.X - a.X*b.Z, a.X*b.Y - a.Y*b.X}
}

func norm(a Vec) Vec {
	l := math.Sqrt(dot(a, a))
	if l == 0 {
		return a
	}
	return Scale(a, 1/l)
}

// mat は行優先の4x4行列。点は列ベクトルとして mat*v で変換する
type mat [16]float64

func mul(a, b mat) mat {
	var c mat
	for i := range 4 {
		for j := range 4 {
			s := 0.0
			for k := range 4 {
				s += a[i*4+k] * b[k*4+j]
			}
			c[i*4+j] = s
		}
	}
	return c
}

func apply(m mat, p Vec) (x, y, z, wc float64) {
	x = m[0]*p.X + m[1]*p.Y + m[2]*p.Z + m[3]
	y = m[4]*p.X + m[5]*p.Y + m[6]*p.Z + m[7]
	z = m[8]*p.X + m[9]*p.Y + m[10]*p.Z + m[11]
	wc = m[12]*p.X + m[13]*p.Y + m[14]*p.Z + m[15]
	return
}

func perspective(fovyDeg, aspect, near, far float64) mat {
	f := 1.0 / math.Tan(fovyDeg*math.Pi/180/2)
	return mat{
		f / aspect, 0, 0, 0,
		0, f, 0, 0,
		0, 0, (far + near) / (near - far), (2 * far * near) / (near - far),
		0, 0, -1, 0,
	}
}

func lookAt(eye, center, up Vec) mat {
	f := norm(sub(center, eye))
	s := norm(cross(f, up))
	u := cross(s, f)
	return mat{
		s.X, s.Y, s.Z, -dot(s, eye),
		u.X, u.Y, u.Z, -dot(u, eye),
		-f.X, -f.Y, -f.Z, dot(f, eye),
		0, 0, 0, 1,
	}
}

// At は座標から点を作る。3成分を並べる式が読みやすくなる
func At(x, y, z float64) Vec { return Vec{X: x, Y: y, Z: z} }
