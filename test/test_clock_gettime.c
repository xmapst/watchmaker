#include <stdio.h>
#include <unistd.h>
#include <time.h>

int main() {
    struct timespec ts_realtime;
    struct tm *tm_info;
    char buffer[26];

    for (int i = 0; i < 10; i++) {
        if (clock_gettime(CLOCK_REALTIME, &ts_realtime) == -1) {
            perror("clock_gettime(CLOCK_REALTIME) failed");
            return 1;
        }

        tm_info = localtime(&ts_realtime.tv_sec);
        if (tm_info == NULL) {
            perror("localtime() failed");
            return 1;
        }

        strftime(buffer, sizeof(buffer), "%Y-%m-%d %H:%M:%S", tm_info);
        printf("[%d] clock_gettime(CLOCK_REALTIME) returns: %s.%09ld\n", getpid(), buffer, ts_realtime.tv_nsec);

        sleep(1);
    }

    return 0;
}
