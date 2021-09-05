### Use "make" or "make debug"

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
LIBRARY = $(LIBDIR)/libtracer.so
STATIC_LIBRARY = $(LIBDIR)/libtracer.a
SRCFILES = $(SRCDIR)/tracer.c $(SRCDIR)/queue.c $(SRCDIR)/buffer.c $(SRCDIR)/tracestate.c $(SRCDIR)/trigger.c $(SRCDIR)/breadcrumb.c $(SRCDIR)/hindsight.c $(SRCDIR)/agentapi.c $(SRCDIR)/common.c

DEPFILES := $(SRCFILES:%.c=$(DEPDIR)/%.d)
DEPFLAGS = -MT $@ -MMD -MP -MF $(DEPDIR)/$*.d
OBJFILES = ${SRCFILES:.c=.o}
INCLUDEFLAGS = -I $(SRCDIR)
PTHREADFLAGS = -lpthread -lm

TARGET = dirs bin/breadcrumbs_test bin/benchmark_test bin/triggers_test bin/buffer_test bin/queue_test bin/tracer_test bin/client_test bin/multi_client_test bin/hindsight_test bin/hindsight2_test bin/data_test bin/sampling_test bin/p99_test $(LIBRARY) $(STATIC_LIBRARY)
.PHONY: all debug

all: $(TARGET)

debug: CFLAGS += -DDEBUG=1 -ggdb3
debug: clean all

dirs:
	@mkdir -p $(BINDIR) $(LIBDIR)

### APPLICATIONS
$(BINDIR)/%_test: $(TESTDIR)/%.c $(OBJFILES)
	$(CC) $(CFLAGS) -o $@ $^ $(INCLUDEFLAGS) $(PTHREADFLAGS) 

### LIBRARIES
$(STATIC_LIBRARY): $(OBJFILES) 
	$(AR) $(ARFLAGS) $@ $(OBJFILES)

$(LIBRARY): $(OBJFILES) 
	$(CC) $(CFLAGS) $(LDFLAGS) -o $@ $(OBJFILES) $(INCLUDEFLAGS) $(PTHREADFLAGS)


### OBJECTS
%.o : %.c $(DEPDIR) Makefile #$(DEPDIR)/%.d
	$(CC) $(CFLAGS) $(STATIC_LDFLAGS) $(INCLUDEFLAGS) $(PTHREADFLAGS) $(DEPFLAGS) -c -o $@ $<

$(DEPDIR): ; @mkdir -p $@ $@/$(SRCDIR) $@/$(TESTDIR)

install:
	rm -rf /usr/local/include/tracer /usr/local/lib/libtracer.so
	cp lib/libtracer.so /usr/local/lib
	cp lib/libtracer.a /usr/local/lib
	cp -r src /usr/local/include/tracer
	mkdir -p /etc/hindsight_conf
	cp ../conf/* /etc/hindsight_conf/

clean:
	rm -rf $(TARGET)
	rm -rf $(SRCDIR)/*.o
