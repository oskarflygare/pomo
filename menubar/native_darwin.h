//go:build darwin && cgo

#include <stdbool.h>

void pomoRun(void);
void pomoQuit(void);
void pomoSetStatus(const char *label, double progress, bool paused);
void pomoMenuBegin(void);
void pomoMenuAdd(int parent, int itemID, const char *title, const char *detail,
                 bool enabled, bool checked, bool separator, bool hasAction);
void pomoMenuShow(void);
