package xcoord

import "math"

// ProjType 投影类型
type ProjType int

const (
	GK  ProjType = iota // 高斯-克吕格(Gauss-Kruger)投影
	UTM                 // UTM投影
)

// EllipsoidParameter 椭球体参数
type EllipsoidParameter struct {
	A   float64
	B   float64
	F   float64
	E2  float64
	EP2 float64
	C   float64
	A0  float64
	A2  float64
	A4  float64
	A6  float64
}

// NewWGS84Parameter 生成 WGS84 椭球体参数
func NewWGS84Parameter() *EllipsoidParameter {
	ep := &EllipsoidParameter{
		A:  6378137.0,
		E2: 0.00669437999013,
	}

	ep.B = math.Sqrt(ep.A * ep.A * (1 - ep.E2))
	ep.EP2 = (ep.A*ep.A - ep.B*ep.B) / (ep.B * ep.B)
	ep.F = (ep.A - ep.B) / ep.A

	// f0 := 1 / 298.257223563;
	// f1 := 1 / ep.F;

	ep.C = ep.A / (1 - ep.F)

	m0 := ep.A * (1 - ep.E2)
	m2 := 1.5 * ep.E2 * m0
	m4 := 1.25 * ep.E2 * m2
	m6 := 7 * ep.E2 * m4 / 6
	m8 := 9 * ep.E2 * m6 / 8

	ep.A0 = m0 + m2/2 + 3*m4/8 + 5*m6/16 + 35*m8/128
	ep.A2 = m2/2 + m4/2 + 15*m6/32 + 7*m8/16
	ep.A4 = m4/8 + 3*m6/16 + 7*m8/32
	ep.A6 = m6/32 + m8/16

	return ep
}

// ZtGeoCoordTransform 经纬度与大地平面直角坐标系间的转换；
//
//	[翻译自C++代码](https://www.cnblogs.com/xingzhensun/p/11377963.html)
type ZtGeoCoordTransform struct {
	ep *EllipsoidParameter
	ml int
	pt ProjType
}

// NewZtGeoCoordTransform 返回经纬度与大地平面直角坐标系间的转换器；
//
//	[示例]
//	zgct := contrib.NewZtGeoCoordTransform(-360, contrib.GK)
func NewZtGeoCoordTransform(ml int, pt ProjType) *ZtGeoCoordTransform {
	return &ZtGeoCoordTransform{
		ep: NewWGS84Parameter(),
		ml: ml,
		pt: pt,
	}
}

// BL2XY 经纬度转大地平面直角坐标系点
func (zgct *ZtGeoCoordTransform) BL2XY(loc *Location) *Point {
	ml := zgct.ml

	if ml < -180 {
		ml = int((loc.lng+1.5)/3) * 3
	}

	lat := loc.lat * 0.0174532925199432957692
	dL := (loc.lng - float64(ml)) * 0.0174532925199432957692

	X := zgct.ep.A0*lat - zgct.ep.A2*math.Sin(2*lat)/2 + zgct.ep.A4*math.Sin(4*lat)/4 - zgct.ep.A6*math.Sin(6*lat)/6

	tn := math.Tan(lat)
	tn2 := tn * tn
	tn4 := tn2 * tn2

	j2 := (1/math.Pow(1-zgct.ep.F, 2) - 1) * math.Pow(math.Cos(lat), 2)
	n := zgct.ep.A / math.Sqrt(1.0-zgct.ep.E2*math.Sin(lat)*math.Sin(lat))

	var temp [6]float64

	temp[0] = n * math.Sin(lat) * math.Cos(lat) * dL * dL / 2
	temp[1] = n * math.Sin(lat) * math.Pow(math.Cos(lat), 3) * (5 - tn2 + 9*j2 + 4*j2*j2) * math.Pow(dL, 4) / 24
	temp[2] = n * math.Sin(lat) * math.Pow(math.Cos(lat), 5) * (61 - 58*tn2 + tn4) * math.Pow(dL, 6) / 720
	temp[3] = n * math.Cos(lat) * dL
	temp[4] = n * math.Pow(math.Cos(lat), 3) * (1 - tn2 + j2) * math.Pow(dL, 3) / 6
	temp[5] = n * math.Pow(math.Cos(lat), 5) * (5 - 18*tn2 + tn4 + 14*j2 - 58*tn2*j2) * math.Pow(dL, 5) / 120

	px := temp[3] + temp[4] + temp[5]
	py := X + temp[0] + temp[1] + temp[2]

	switch zgct.pt {
	case GK:
		px += 500000
	case UTM:
		px = px*0.9996 + 500000
		py = py * 0.9996
	}

	return NewPoint(px, py, ml)
}

// XY2BL 大地平面直角坐标系点转经纬度
func (zgct *ZtGeoCoordTransform) XY2BL(p *Point) *Location {
	x := p.x - 500000
	y := p.y

	if zgct.pt == UTM {
		x = x / 0.9996
		y = y / 0.9996
	}

	var (
		bf0       = y / zgct.ep.A0
		bf        float64
		threshold = 1.0
	)

	for threshold > 0.00000001 {
		y0 := -zgct.ep.A2*math.Sin(2*bf0)/2 + zgct.ep.A4*math.Sin(4*bf0)/4 - zgct.ep.A6*math.Sin(6*bf0)/6
		bf = (y - y0) / zgct.ep.A0

		threshold = bf - bf0
		bf0 = bf
	}

	t := math.Tan(bf)
	j2 := zgct.ep.EP2 * math.Pow(math.Cos(bf), 2)

	v := math.Sqrt(1 - zgct.ep.E2*math.Sin(bf)*math.Sin(bf))
	n := zgct.ep.A / v
	m := zgct.ep.A * (1 - zgct.ep.E2) / math.Pow(v, 3)

	temp0 := t * x * x / (2 * m * n)
	temp1 := t * (5 + 3*t*t + j2 - 9*j2*t*t) * math.Pow(x, 4) / (24 * m * math.Pow(n, 3))
	temp2 := t * (61 + 90*t*t + 45*math.Pow(t, 4)) * math.Pow(x, 6) / (720 * math.Pow(n, 5) * m)

	lat := (bf - temp0 + temp1 - temp2) * 57.29577951308232

	temp0 = x / (n * math.Cos(bf))
	temp1 = (1 + 2*t*t + j2) * math.Pow(x, 3) / (6 * math.Pow(n, 3) * math.Cos(bf))
	temp2 = (5 + 28*t*t + 6*j2 + 24*math.Pow(t, 4) + 8*t*t*j2) * math.Pow(x, 5) / (120 * math.Pow(n, 5) * math.Cos(bf))

	lng := (temp0-temp1+temp2)*57.29577951308232 + float64(p.ml)

	return NewLocation(lng, lat)
}
