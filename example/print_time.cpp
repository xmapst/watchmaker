#include <iostream>
#include <thread>
#include <mutex>
#include <condition_variable>
#include <chrono>
#include <ctime>
#include <sys/time.h>
#include <unistd.h>
#include <iomanip>
#include <sstream>
#include <array>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <fcntl.h>
#include <cstring>
#include <signal.h>

#ifdef __APPLE__
#include <sys/event.h>
#else
#include <sys/epoll.h>
#endif

// 获取格式化时间的函数
template<typename TimeFunc>
std::string getFormattedTime(TimeFunc timeFunc) {
    return timeFunc();
}

auto clockGetTimeFunc = []() -> std::string {
    timespec ts;
    clock_gettime(CLOCK_REALTIME, &ts);
    tm tm_ts;
    localtime_r(&ts.tv_sec, &tm_ts);
    std::ostringstream oss;
    oss << std::put_time(&tm_ts, "%Y-%m-%d %H:%M:%S")
        << '.' << std::setw(9) << std::setfill('0') << ts.tv_nsec;
    return oss.str();
};

auto timeFunc = []() -> std::string {
    time_t t = time(nullptr);
    tm tm_time;
    localtime_r(&t, &tm_time);
    std::ostringstream oss;
    oss << std::put_time(&tm_time, "%Y-%m-%d %H:%M:%S");
    return oss.str();
};

auto gettimeofdayFunc = []() -> std::string {
    timeval tv;
    gettimeofday(&tv, nullptr);
    tm tm_tv;
    localtime_r(&tv.tv_sec, &tm_tv);
    std::ostringstream oss;
    oss << std::put_time(&tm_tv, "%Y-%m-%d %H:%M:%S")
        << '.' << std::setw(6) << std::setfill('0') << tv.tv_usec;
    return oss.str();
};

// epoll/kqueue server thread function (returns formatted time JSON, sent once per second)
void serverThread() {
    const int PORT = 8080;
    const int MAX_EVENTS = 10;

#ifdef __APPLE__
    int kq = kqueue();
    if (kq == -1) {
        perror("kqueue");
        return;
    }
#else
    int epoll_fd = epoll_create1(0);
    if (epoll_fd == -1) {
        perror("epoll_create1");
        return;
    }
#endif

    int listen_sock = socket(AF_INET, SOCK_STREAM, 0);
    if (listen_sock == -1) {
        perror("socket");
        return;
    }

    if (fcntl(listen_sock, F_SETFL, O_NONBLOCK) == -1) {
        perror("fcntl O_NONBLOCK listen_sock");
        close(listen_sock);
        return;
    }

    int optval = 1;
    setsockopt(listen_sock, SOL_SOCKET, SO_REUSEADDR, &optval, sizeof(optval));

    sockaddr_in server_addr{};
    server_addr.sin_family = AF_INET;
    server_addr.sin_addr.s_addr = INADDR_ANY;
    server_addr.sin_port = htons(PORT);

    if (bind(listen_sock, (sockaddr*)&server_addr, sizeof(server_addr)) == -1) {
        perror("bind");
        close(listen_sock);
        return;
    }

    if (listen(listen_sock, SOMAXCONN) == -1) {
        perror("listen");
        close(listen_sock);
        return;
    }

#ifdef __APPLE__
    struct kevent change;
    EV_SET(&change, listen_sock, EVFILT_READ, EV_ADD | EV_ENABLE, 0, 0, nullptr);
    if (kevent(kq, &change, 1, nullptr, 0, nullptr) == -1) {
        perror("kevent add listen_sock");
        close(listen_sock);
        close(kq);
        return;
    }
    struct kevent event_list[MAX_EVENTS];
#else
    epoll_event event{};
    event.events = EPOLLIN;
    event.data.fd = listen_sock;
    if (epoll_ctl(epoll_fd, EPOLL_CTL_ADD, listen_sock, &event) == -1) {
        perror("epoll_ctl");
        close(listen_sock);
        close(epoll_fd);
        return;
    }

    epoll_event events[MAX_EVENTS];
#endif

    while (true) {
#ifdef __APPLE__
        int num_events = kevent(kq, nullptr, 0, event_list, MAX_EVENTS, nullptr);
        if (num_events == -1) {
            if (errno == EINTR)
                continue;
            perror("kevent");
            break;
        }

        for (int i = 0; i < num_events; ++i) {
            if (event_list[i].ident == (uintptr_t)listen_sock) {
                sockaddr_in client_addr{};
                socklen_t client_len = sizeof(client_addr);
                int client_fd = accept(listen_sock, (sockaddr*)&client_addr, &client_len);
                if (client_fd == -1) {
                    if (errno != EAGAIN && errno != EWOULDBLOCK)
                        perror("accept");
                    continue;
                }

                if (fcntl(client_fd, F_SETFL, O_NONBLOCK) == -1) {
                    perror("fcntl O_NONBLOCK client_fd");
                    close(client_fd);
                    continue;
                }

                EV_SET(&change, client_fd, EVFILT_READ, EV_ADD | EV_ENABLE, 0, 0, nullptr);
                if (kevent(kq, &change, 1, nullptr, 0, nullptr) == -1) {
                    perror("kevent add client_fd");
                    close(client_fd);
                    continue;
                }

                // start a thread for each connection and send the time JSON once a second
                std::thread([client_fd]() {
                    while (true) {
                        std::ostringstream json_response;
                        json_response << "{\"pid\": " << getpid() << ",";
                        json_response << "\"clock_gettime\": \"" << clockGetTimeFunc() << "\",";
                        json_response << "\"time\": \"" << timeFunc() << "\",";
                        json_response << "\"gettimeofday\": \"" << gettimeofdayFunc() << "\"}\n";
                        std::string response = json_response.str();
                        ssize_t sent = send(client_fd, response.c_str(), response.size(), 0);
                        if (sent == -1) {
                            perror("send");
                            break;
                        }
                        std::this_thread::sleep_for(std::chrono::seconds(1));
                    }
                    close(client_fd);
                }).detach();
            } else if (event_list[i].flags & EV_EOF) {
                close(event_list[i].ident);
                std::cerr << "Client disconnected via EV_EOF: " << event_list[i].ident << std::endl;
            }
        }
#else
        int num_events = epoll_wait(epoll_fd, events, MAX_EVENTS, -1);
        if (num_events == -1) {
            if (errno == EINTR)
                continue;
            perror("epoll_wait");
            break;
        }
        for (int i = 0; i < num_events; ++i) {
            if (events[i].data.fd == listen_sock) {
                sockaddr_in client_addr{};
                socklen_t client_len = sizeof(client_addr);
                int client_fd = accept(listen_sock, (sockaddr*)&client_addr, &client_len);
                if (client_fd == -1) {
                    if (errno != EAGAIN && errno != EWOULDBLOCK)
                        perror("accept");
                    continue;
                }
                // 设置非阻塞模式
                fcntl(client_fd, F_SETFL, O_NONBLOCK);

                // 为每个连接启动一个线程，每秒发送一次时间 JSON
                std::thread([client_fd]() {
                    while (true) {
                        std::ostringstream json_response;
                        json_response << "{\"pid\": " << getpid() << ",";
                        json_response << "\"clock_gettime\": \"" << clockGetTimeFunc() << "\",";
                        json_response << "\"time\": \"" << timeFunc() << "\",";
                        json_response << "\"gettimeofday\": \"" << gettimeofdayFunc() << "\"}\n";
                        std::string response = json_response.str();
                        ssize_t sent = send(client_fd, response.c_str(), response.size(), 0);
                        if (sent == -1) {
                            perror("send");
                            break;
                        }
                        std::this_thread::sleep_for(std::chrono::seconds(1));
                    }
                    close(client_fd);
                }).detach();
            }
        }
#endif
    }
    close(listen_sock);
#ifdef __APPLE__
    close(kq);
#else
    close(epoll_fd);
#endif
}

int main() {
    struct sigaction sa;
    sa.sa_handler = SIG_IGN;
    sigaction(SIGPIPE, &sa, nullptr);

    std::thread server_thread(serverThread);
    server_thread.detach();
    // 主线程每秒打印三种时间格式
    while (true) {
        std::cout << "[" << getpid() << "] clock_gettime: " << clockGetTimeFunc()
                  << ", time: " << timeFunc()
                  << ", gettimeofday: " << gettimeofdayFunc() << std::endl;
        sleep(1);
    }
    return 0;
}
