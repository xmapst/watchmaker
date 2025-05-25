#include <sys/time.h>
#include <stdio.h>
#include <unistd.h>
#include <time.h>

int main() {
    struct timeval tv;
    struct tm *tm_info;
    char buffer[26];

    while (1) {
        if (gettimeofday(&tv, NULL) == -1) {
            perror("gettimeofday() failed");
            return 1;
        }

        tm_info = localtime(&tv.tv_sec);

        if (tm_info == NULL) {
            perror("localtime() failed");
            return 1;
        }

        strftime(buffer, sizeof(buffer), "%Y-%m-%d %H:%M:%S", tm_info);

        printf("[%d] gettimeofday() returns: %s.%06ld\n", getpid(), buffer, tv.tv_usec);
        sleep(1);
    }

    return 0;
}
