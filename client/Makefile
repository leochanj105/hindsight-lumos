### Use "make"

CC = gcc
AR = ar
ARFLAGS = rcs

LIBDIR = lib
SRCDIR = src
BINDIR = bin
TESTDIR = test
JNIDIR = jni
DEPDIR = .deps

CFLAGS = -Wall -rdynamic -O3 -pedantic
STATIC_LDFLAGS = -fPIC
LDFLAGS = $(STATIC_LDFLAGS) -shared

SRCFILES = $(SRCDIR)/tracer.c $(SRCDIR)/queue.c \
			$(SRCDIR)/memory.c
DEPFILES := $(SRCFILES:%.c=$(DEPDIR)/%.d)
DEPFLAGS = -MT $@ -MMD -MP -MF $(DEPDIR)/$*.d
OBJFILES = ${SRCFILES:.c=.o}
INCLUDEFLAGS = -I $(SRCDIR)
PTHREADFLAGS = -lpthread

TARGET = dirs bin/basic_test
.PHONY: all

all: $(TARGET)


dirs:
	@mkdir -p $(BINDIR)

### APPLICATIONS
$(BINDIR)/%_test: $(TESTDIR)/%.c $(OBJFILES)
	$(CC) $(CFLAGS) -o $@ $^ $(INCLUDEFLAGS) $(PTHREADFLAGS) 

### OBJECTS
%.o : %.c $(DEPDIR) Makefile #$(DEPDIR)/%.d
	$(CC) $(CFLAGS) $(STATIC_LDFLAGS) $(INCLUDEFLAGS) $(PTHREADFLAGS) $(DEPFLAGS) -c -o $@ $<

clean:
	rm -rf $(TARGET)