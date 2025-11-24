# Makefile for building NIST SP800-90B C++ Library

ARCH ?= x86
CXX ?= $(CROSS_COMPILE)g++

CXXFLAGS = -std=c++11 -fopenmp -O2 -ffloat-store -I/usr/include/jsoncpp
ifeq ($(ARCH),x86)
CXXFLAGS += -march=native
endif

LIB = -lbz2 -lpthread -ldivsufsort -ldivsufsort64
COND_LIB = -lmpfr -lgmp
SHARED_LIB = -ljsoncpp -lcrypto

# Directories  
CPP_DIR = cpp
WRAPPER_DIR = wrapper
LIB_DIR = lib

# Object files
WRAPPER_OBJ = $(WRAPPER_DIR)/wrapper.o

# Library output
STATIC_LIB = $(LIB_DIR)/libentropy90b.a

.PHONY: all clean lib

all: lib

lib: $(STATIC_LIB)

# Create static library from wrapper object
$(STATIC_LIB): $(WRAPPER_OBJ)
	@mkdir -p $(LIB_DIR)
	$(AR) rcs $@ $(WRAPPER_OBJ)

# Compile wrapper (includes NIST headers which are mostly header-only)
$(WRAPPER_DIR)/wrapper.o: $(WRAPPER_DIR)/wrapper.cpp $(WRAPPER_DIR)/wrapper.h
	$(CXX) $(CXXFLAGS) -I$(CPP_DIR) -c $< -o $@

clean:
	rm -f $(WRAPPER_OBJ) $(STATIC_LIB)
	rm -rf $(LIB_DIR)

help:
	@echo "Makefile for NIST SP800-90B C++ Library"
	@echo ""
	@echo "Targets:"
	@echo "  all (default) - Build static library"
	@echo "  lib           - Build static library"
	@echo "  clean         - Remove all build artifacts"
	@echo "  help          - Show this help message"
	@echo ""
	@echo "Variables:"
	@echo "  ARCH          - Architecture (x86, aarch64, etc.) [default: x86]"
	@echo "  CROSS_COMPILE - Cross-compiler prefix [default: none]"
	@echo "  CXX           - C++ compiler [default: g++]"
