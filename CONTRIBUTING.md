# Contributing to NepKey

## Clone & build

```bash
git clone https://github.com/todo-laterx/nepkey.git
cd nepkey
go vet ./...
go test ./...
go build -o /tmp/nepkey-ibus ./cmd/nepkey-ibus
bash scripts/install.sh
```

## Quy tac MR

1. Fork repo, tao nhanh `feat/<ten>`, `fix/<ten>`, khong push thang `main`.
2. Moi thay doi `pkg/ibus/engine.go` bat buoc kem test trong `pkg/ibus/engine_test.go` hoac `digit_regression_test.go`.
3. Mo ta MR theo mau: van de / cach tai hien / app test (Terminal, VS Code, Chrome) tren Wayland hay X11 / ket qua `go test ./...`.
4. Khong commit binary, `.deb`, `__pycache__`, `dist/`, `build/deb/`. Chi track source.
5. Neu them thuoc tinh metadata/column moi cho FE, dong bo schema backend tuong ung (nguyen tac twin-component parity).

## Bao loi

Ghi ro: Ubuntu version, GNOME Wayland/X11, IBus version, app gay loi, phim bam tung buoc, ky vong vs thuc te, log `journalctl --user -u ibus*` neu co.
