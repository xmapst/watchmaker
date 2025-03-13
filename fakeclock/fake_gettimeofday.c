#include <time.h>
#include <sys/time.h>
#include <inttypes.h>
#include <syscall.h>

extern int64_t TV_SEC_DELTA;
extern int64_t TV_NSEC_DELTA;

#if defined(__amd64__)
inline int real_gettimeofday(struct timeval *tv, struct timezone *tz)
{
    int ret;
    asm volatile(
        "syscall"
        : "=a"(ret)
        : "0"(__NR_gettimeofday), "D"(tv), "S"(tz)
        : "rcx", "r11", "memory"
    );
    return ret;
}
#elif defined(__aarch64__)
inline int real_gettimeofday(struct timeval *tv, struct timezone *tz)
{
    register int w0 __asm__("w0");
    register struct timeval *x0 __asm__("x0") = tv;
    register struct timezone *x1 __asm__("x1") = tz;
    register uint64_t w8 __asm__("w8") = __NR_gettimeofday;
    __asm__ __volatile__(
        "svc 0"
        : "+r"(w0)
        : "r"(x0), "r"(x1), "r"(w8)
        : "memory"
    );
    return w0;
}
#endif

int fake_gettimeofday(struct timeval *tv, struct timezone *tz)
{
    int ret = real_gettimeofday(tv, tz);

    const int64_t sec_delta = TV_SEC_DELTA;
    const int64_t nsec_delta = TV_NSEC_DELTA;
    const int64_t billion = 1000000000;

    // 计算总纳秒偏移量（当前微秒转换为纳秒 + 纳秒偏移）
    int64_t total_nsec = (int64_t)tv->tv_usec * 1000 + nsec_delta;

    // 分解为秒和剩余纳秒
    int64_t extra_sec = total_nsec / billion;
    int64_t remaining_nsec = total_nsec % billion;

    // 处理负剩余纳秒
    if (remaining_nsec < 0) {
        extra_sec -= 1;
        remaining_nsec += billion;
    }

    // 四舍五入到最近的微秒
    int64_t usec_adjust = (remaining_nsec + 500) / 1000;

    // 更新秒和微秒
    tv->tv_sec += sec_delta + extra_sec;
    tv->tv_usec += usec_adjust;

    // 处理微秒溢出/下溢
    if (tv->tv_usec >= 1000000) {
        tv->tv_sec += 1;
        tv->tv_usec -= 1000000;
    } else if (tv->tv_usec < 0) {
        tv->tv_sec -= 1;
        tv->tv_usec += 1000000;
    }

    return ret;
}