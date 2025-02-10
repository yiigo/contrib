package yiigo

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
)

type Step struct {
	Head int
	Tail int
}

// Steps calculates the steps.
//
// Example:
//
//	arr := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20}
//	for _, step := range yiigo.Steps(len(arr), 6) {
//		cur := arr[step.Head:step.Tail]
//		// todo: do something
//	}
func Steps(total, step int) (steps []Step) {
	steps = make([]Step, 0, int(math.Ceil(float64(total)/float64(step))))
	for i := 0; i < total; i++ {
		if i%step == 0 {
			head := i
			tail := head + step
			if tail > total {
				tail = total
			}
			steps = append(steps, Step{Head: head, Tail: tail})
		}
	}
	return steps
}

// Nonce 生成随机串(size应为偶数)
func Nonce(size uint8) string {
	nonce := make([]byte, size/2)
	_, _ = io.ReadFull(rand.Reader, nonce)
	return hex.EncodeToString(nonce)
}

// MarshalNoEscapeHTML 不带HTML转义的JSON序列化
func MarshalNoEscapeHTML(v any) ([]byte, error) {
	buf := bytes.NewBuffer(nil)

	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(v); err != nil {
		return nil, err
	}

	// 去掉 go std 给末尾加的 '\n'
	// @see https://github.com/golang/go/issues/7767
	b := buf.Bytes()
	if l := len(b); l != 0 && b[l-1] == '\n' {
		b = b[:l-1]
	}
	return b, nil
}

// VersionCompare 语义化的版本比较，支持：>, >=, =, !=, <, <=, | (or), & (and).
// 参数 `rangeVer` 示例：1.0.0, =1.0.0, >2.0.0, >=1.0.0&<2.0.0, <2.0.0|>3.0.0, !=4.0.4
func VersionCompare(rangeVer, curVer string) (bool, error) {
	semVer, err := version.NewVersion(curVer)
	if err != nil {
		return false, err
	}

	orVers := strings.Split(rangeVer, "|")
	for _, ver := range orVers {
		andVers := strings.Split(ver, "&")
		constraints, err := version.NewConstraint(strings.Join(andVers, ","))
		if err != nil {
			return false, err
		}
		if constraints.Check(semVer) {
			return true, nil
		}
	}
	return false, nil
}

// Retry 重试
func Retry(ctx context.Context, fn func(ctx context.Context) error, attempts int, sleep time.Duration) (err error) {
	threshold := attempts - 1
	for i := 0; i < attempts; i++ {
		err = fn(ctx)
		if err == nil || i >= threshold {
			return
		}
		time.Sleep(sleep)
	}
	return
}

// IsUniqueDuplicateError 判断是否「唯一索引冲突」错误
func IsUniqueDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	for _, s := range []string{
		"Duplicate entry",            // MySQL
		"violates unique constraint", // Postgres
		"UNIQUE constraint failed",   // SQLite
	} {
		if strings.Contains(err.Error(), s) {
			return true
		}
	}
	return false
}

// ExcelColumnIndex 返回Excel列名对应的序号，如：A=0，B=1，AA=26，AB=27
func ExcelColumnIndex(name string) int {
	name = strings.ToUpper(name)
	if ok, err := regexp.MatchString(`^[A-Z]{1,2}$`, name); err != nil || !ok {
		return -1
	}

	index := 0
	for i, v := range name {
		if i != 0 {
			index = (index + 1) * 26
		}
		index += int(v - 'A')
	}
	return index
}
