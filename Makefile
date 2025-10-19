# Makefile for go-randomx development
.PHONY: help test test-vectors test-comparison build-cpp-trace generate-cpp-traces clean

help:
	@echo "go-randomx Development Commands:"
	@echo ""
	@echo "  make test                 - Run all Go tests"
	@echo "  make test-vectors         - Run official RandomX test vectors"
	@echo "  make test-comparison      - Run C++ reference comparison tests"
	@echo "  make test-debug           - Run tests with debug tracing enabled"
	@echo "  make build-cpp-trace      - Build C++ trace extraction tool"
	@echo "  make generate-cpp-traces  - Generate reference traces from C++"
	@echo "  make clean                - Clean build artifacts"
	@echo ""

# Run all Go tests
test:
	go test -v ./...

# Run official test vectors specifically
test-vectors:
	go test -v -run TestOfficialVectors

# Run C++ reference comparison tests
test-comparison:
	go test -v -run TestCompareWithCPPReference

# Run tests with debug tracing enabled
test-debug:
	RANDOMX_DEBUG=1 go test -v -run TestOfficialVectors/basic_test_1

# Build the C++ trace extraction tool
build-cpp-trace:
	@echo "Building C++ trace extractor..."
	@if [ ! -d "tools/cpp_trace_extractor/build" ]; then \
		mkdir -p tools/cpp_trace_extractor/build; \
	fi
	@cd tools/cpp_trace_extractor/build && cmake .. && make
	@echo "✓ C++ trace extractor built: tools/cpp_trace_extractor/build/extract_trace"

# Generate reference traces from C++ implementation
generate-cpp-traces: build-cpp-trace
	@echo "Generating reference traces from C++ RandomX..."
	@mkdir -p testdata/reference_traces
	@./tools/cpp_trace_extractor/build/extract_trace \
		"test key 000" \
		"This is a test" \
		> testdata/reference_traces/basic_test_1.json
	@./tools/cpp_trace_extractor/build/extract_trace \
		"test key 000" \
		"Lorem ipsum dolor sit amet" \
		> testdata/reference_traces/basic_test_2.json
	@./tools/cpp_trace_extractor/build/extract_trace \
		"test key 000" \
		"sed do eiusmod tempor incididunt ut labore et dolore magna aliqua" \
		> testdata/reference_traces/basic_test_3.json
	@./tools/cpp_trace_extractor/build/extract_trace \
		"test key 001" \
		"sed do eiusmod tempor incididunt ut labore et dolore magna aliqua" \
		> testdata/reference_traces/different_key.json
	@echo "✓ Reference traces generated in testdata/reference_traces/"

# Clean build artifacts
clean:
	rm -rf tools/cpp_trace_extractor/build
	rm -rf testdata/reference_traces
	go clean
	@echo "✓ Build artifacts cleaned"
