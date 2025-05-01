#include <time.h>
#include <sys/time.h>
#include <inttypes.h>
#include <syscall.h>

extern int64_t TV_SEC_DELTA;
extern int64_t TV_NSEC_DELTA;

#if defined(__amd64__)
inline time_t real_time(time_t *t) {
    long ret;
    asm volatile (
        "syscall"
        : "=a"(ret)
        : "0"(__NR_time), "D"(t)
        : "rcx", "r11", "memory"
    );
    return (time_t)ret;
}
#elif defined(__aarch64__)
// apparently this is wrong as it taken from x86_64-linux-gnu/asm/unistd_64.h
// of linux-libc-dev:amd64
#ifndef __NR_time
#define __NR_time 201
#endif
inline time_t real_time(time_t *t) {
    register long ret __asm__("x0") = (long)t;
    register long x8 __asm__("x8") = __NR_time;
    __asm__ volatile(
        "svc 0"
        : "+r"(ret)
        : "r"(x8)
        : "memory"
    );
    return (time_t)ret;
}
#endif

time_t fake_time(time_t *t) {
    time_t original_time = real_time(t);

    const int64_t sec_delta = TV_SEC_DELTA;
    const int64_t nsec_delta = TV_NSEC_DELTA;
    const int64_t billion = 1000000000;

    // 计算额外秒数和剩余纳秒
    int64_t extra_sec = nsec_delta / billion;
    int64_t remaining_nsec = nsec_delta % billion;
    if (remaining_nsec < 0) {
        extra_sec -= 1;
        remaining_nsec += billion;
    }

    // 四舍五入到最近的秒
    if (remaining_nsec >= 500000000) {
        extra_sec += 1;
    }

    // 计算最终时间
    time_t modified_time = original_time + sec_delta + extra_sec;

    if (t) {
        *t = modified_time;
    }

    return modified_time;
}
