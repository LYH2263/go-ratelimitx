package ratelimitx

import "errors"

var (
	// ErrInvalidN 表示 Allow/AllowN 的 n 非法（必须为正）。
	ErrInvalidN = errors.New("ratelimitx: n must be positive")

	// ErrInvalidUnits 表示配额单位非法（必须为正）。
	ErrInvalidUnits = errors.New("ratelimitx: units must be positive")

	// ErrUnknownPolicy 表示未注册的策略名。
	ErrUnknownPolicy = errors.New("ratelimitx: unknown policy")

	// ErrDuplicatePolicy 表示同名策略已存在。
	ErrDuplicatePolicy = errors.New("ratelimitx: duplicate policy name")

	// ErrInvalidSpec 表示策略规格不完整或字段互斥失败。
	ErrInvalidSpec = errors.New("ratelimitx: invalid policy spec")

	// ErrQuotaExceeded 表示硬限额已满且未开启负债模式。
	ErrQuotaExceeded = errors.New("ratelimitx: quota exceeded")

	// ErrUnknownReceipt 表示退款收据不存在。
	ErrUnknownReceipt = errors.New("ratelimitx: unknown receipt")

	// ErrReceiptUsed 表示收据已退过，禁止双花。
	ErrReceiptUsed = errors.New("ratelimitx: receipt already refunded")

	// ErrReceiptMismatch 表示收据与租户不匹配。
	ErrReceiptMismatch = errors.New("ratelimitx: receipt tenant mismatch")

	// ErrImpossible 表示 n 大于突发/窗口上限，永远无法通过。
	ErrImpossible = errors.New("ratelimitx: request exceeds burst or window limit")

	// ErrEmptyKey 表示键为空。
	ErrEmptyKey = errors.New("ratelimitx: empty key")

	// ErrBadDimension 表示复合键维度含有分隔符。
	ErrBadDimension = errors.New("ratelimitx: dimension contains delimiter")

	// ErrZeroShards 表示分片数非法。
	ErrZeroShards = errors.New("ratelimitx: shard count must be at least 1")

	// ErrUnknownAlgorithm 表示不支持的算法名。
	ErrUnknownAlgorithm = errors.New("ratelimitx: unknown algorithm")
)

// ImpossibleWait 表示该请求按当前策略永远无法成功（例如 n > burst）。
const ImpossibleWait = DurationSentinel

// DurationSentinel 用负时长标记“不可能成功”，与真实等待时长区分。
const DurationSentinel = -1
