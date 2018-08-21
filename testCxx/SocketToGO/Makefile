APP = SocketToGo

CPPFLAGS += -std=c++11
LDLIBS += -lboost_system
SRC += SocketToGo.cxx

ALL:
	$(CXX) $(CPPFLAGS) $(SRC) -o $(APP) $(LDLIBS)

clean:
	-rm -f $(APP) *.o

