# just using make as a task runner
.PHONY = bench escape run

bench: # run benchmarks
	go test -bench . -benchmem

escape: # do escape analysis during build
	go build -gcflags '-m' -o /dev/null

run: # run main()
	go run ./