// WASI Preview 1 shim for running Go wasip1/wasm modules in the browser.
// Implements the wasi_snapshot_preview1 import interface.

class WASIExitError extends Error {
    constructor(code) {
        super(`WASI exit with code ${code}`);
        this.code = code;
    }
}

class WASIShim {
    constructor(options = {}) {
        this.stdin = options.stdin || new Uint8Array(0);
        this.stdinOffset = 0;
        this.stdoutChunks = [];
        this.stderrChunks = [];
        this.args = options.args || ['forge-ui-wasm'];
        this.env = options.env || [];
        this.exitCode = null;
        this.instance = null;
    }

    get memory() {
        return this.instance.exports.memory;
    }

    getImportObject() {
        const self = this;
        return {
            wasi_snapshot_preview1: {
                fd_write(fd, iovs, iovsLen, nwritten) {
                    return self._fdWrite(fd, iovs, iovsLen, nwritten);
                },
                fd_read(fd, iovs, iovsLen, nread) {
                    return self._fdRead(fd, iovs, iovsLen, nread);
                },
                fd_close() { return 0; },
                fd_seek(fd, offsetLo, offsetHi, whence, newOffset) {
                    // Return ENOSYS for unseekable fds
                    return 70;
                },
                fd_fdstat_get(fd, buf) {
                    return self._fdFdstatGet(fd, buf);
                },
                fd_prestat_get() { return 8; }, // EBADF - no preopened dirs
                fd_prestat_dir_name() { return 8; },
                path_open() { return 44; }, // ENOENT
                args_get(argv, argvBuf) {
                    return self._argsGet(argv, argvBuf);
                },
                args_sizes_get(argc, argvBufSize) {
                    return self._argsSizesGet(argc, argvBufSize);
                },
                environ_get(environ, environBuf) {
                    return self._environGet(environ, environBuf);
                },
                environ_sizes_get(environc, environBufSize) {
                    return self._environSizesGet(environc, environBufSize);
                },
                clock_time_get(clockId, precisionLo, precisionHi, time) {
                    return self._clockTimeGet(clockId, time);
                },
                proc_exit(code) {
                    self.exitCode = code;
                    throw new WASIExitError(code);
                },
                random_get(buf, bufLen) {
                    return self._randomGet(buf, bufLen);
                },
                sched_yield() { return 0; },
                poll_oneoff(inPtr, outPtr, nsubscriptions, nevents) {
                    return self._pollOneoff(inPtr, outPtr, nsubscriptions, nevents);
                },
            }
        };
    }

    start(instance) {
        this.instance = instance;
        try {
            instance.exports._start();
        } catch (e) {
            if (e instanceof WASIExitError) {
                return e.code;
            }
            throw e;
        }
        return 0;
    }

    getStdout() {
        return this._concatChunks(this.stdoutChunks);
    }

    getStderr() {
        return this._concatChunks(this.stderrChunks);
    }

    _concatChunks(chunks) {
        const totalLen = chunks.reduce((sum, c) => sum + c.length, 0);
        const result = new Uint8Array(totalLen);
        let offset = 0;
        for (const chunk of chunks) {
            result.set(chunk, offset);
            offset += chunk.length;
        }
        return new TextDecoder().decode(result);
    }

    // fd_write: write to stdout (fd=1) or stderr (fd=2)
    _fdWrite(fd, iovs, iovsLen, nwritten) {
        const view = new DataView(this.memory.buffer);
        let totalWritten = 0;

        for (let i = 0; i < iovsLen; i++) {
            const ptr = view.getUint32(iovs + i * 8, true);
            const len = view.getUint32(iovs + i * 8 + 4, true);
            const data = new Uint8Array(this.memory.buffer, ptr, len);

            if (fd === 1) {
                this.stdoutChunks.push(new Uint8Array(data));
            } else if (fd === 2) {
                this.stderrChunks.push(new Uint8Array(data));
            }
            totalWritten += len;
        }

        view.setUint32(nwritten, totalWritten, true);
        return 0;
    }

    // fd_read: read from stdin (fd=0)
    _fdRead(fd, iovs, iovsLen, nread) {
        if (fd !== 0) return 8; // EBADF

        const view = new DataView(this.memory.buffer);
        let totalRead = 0;

        for (let i = 0; i < iovsLen; i++) {
            const ptr = view.getUint32(iovs + i * 8, true);
            const len = view.getUint32(iovs + i * 8 + 4, true);
            const remaining = this.stdin.length - this.stdinOffset;
            const toRead = Math.min(len, remaining);

            if (toRead > 0) {
                const buf = new Uint8Array(this.memory.buffer, ptr, toRead);
                buf.set(this.stdin.subarray(this.stdinOffset, this.stdinOffset + toRead));
                this.stdinOffset += toRead;
                totalRead += toRead;
            }
            if (toRead < len) break;
        }

        view.setUint32(nread, totalRead, true);
        return 0;
    }

    // fd_fdstat_get: return file type for known fds
    _fdFdstatGet(fd, buf) {
        if (fd > 2) return 8; // EBADF
        const view = new DataView(this.memory.buffer);
        // filetype: CHARACTER_DEVICE (2)
        view.setUint8(buf, 2);
        // fdflags: 0
        view.setUint16(buf + 2, 0, true);
        // rights_base: all rights
        view.setBigUint64(buf + 8, BigInt(0), true);
        // rights_inheriting: all rights
        view.setBigUint64(buf + 16, BigInt(0), true);
        return 0;
    }

    // args_get: write argument pointers and strings
    _argsGet(argv, argvBuf) {
        const view = new DataView(this.memory.buffer);
        const encoder = new TextEncoder();
        let bufOffset = argvBuf;

        for (let i = 0; i < this.args.length; i++) {
            view.setUint32(argv + i * 4, bufOffset, true);
            const encoded = encoder.encode(this.args[i] + '\0');
            const target = new Uint8Array(this.memory.buffer, bufOffset, encoded.length);
            target.set(encoded);
            bufOffset += encoded.length;
        }
        return 0;
    }

    // args_sizes_get: return argc and total argv buffer size
    _argsSizesGet(argc, argvBufSize) {
        const view = new DataView(this.memory.buffer);
        const encoder = new TextEncoder();
        let totalSize = 0;
        for (const arg of this.args) {
            totalSize += encoder.encode(arg + '\0').length;
        }
        view.setUint32(argc, this.args.length, true);
        view.setUint32(argvBufSize, totalSize, true);
        return 0;
    }

    // environ_get: write environment variable pointers and strings
    _environGet(environ, environBuf) {
        const view = new DataView(this.memory.buffer);
        const encoder = new TextEncoder();
        let bufOffset = environBuf;

        for (let i = 0; i < this.env.length; i++) {
            view.setUint32(environ + i * 4, bufOffset, true);
            const encoded = encoder.encode(this.env[i] + '\0');
            const target = new Uint8Array(this.memory.buffer, bufOffset, encoded.length);
            target.set(encoded);
            bufOffset += encoded.length;
        }
        return 0;
    }

    // environ_sizes_get: return number of env vars and total buffer size
    _environSizesGet(environc, environBufSize) {
        const view = new DataView(this.memory.buffer);
        const encoder = new TextEncoder();
        let totalSize = 0;
        for (const e of this.env) {
            totalSize += encoder.encode(e + '\0').length;
        }
        view.setUint32(environc, this.env.length, true);
        view.setUint32(environBufSize, totalSize, true);
        return 0;
    }

    // clock_time_get: return current time in nanoseconds
    _clockTimeGet(clockId, timePtr) {
        const view = new DataView(this.memory.buffer);
        const now = BigInt(Date.now()) * BigInt(1000000); // ms to ns
        view.setBigUint64(timePtr, now, true);
        return 0;
    }

    // random_get: fill buffer with random bytes
    _randomGet(buf, bufLen) {
        const buffer = new Uint8Array(this.memory.buffer, buf, bufLen);
        crypto.getRandomValues(buffer);
        return 0;
    }

    // poll_oneoff: minimal implementation that returns immediately
    _pollOneoff(inPtr, outPtr, nsubscriptions, neventsPtr) {
        const view = new DataView(this.memory.buffer);
        // Write 0 events and return success
        view.setUint32(neventsPtr, nsubscriptions, true);
        // Fill output events
        for (let i = 0; i < nsubscriptions; i++) {
            const outOffset = outPtr + i * 32;
            // userdata (8 bytes) - copy from input
            const inOffset = inPtr + i * 48;
            const userdata = view.getBigUint64(inOffset, true);
            view.setBigUint64(outOffset, userdata, true);
            // error (2 bytes) - success
            view.setUint16(outOffset + 8, 0, true);
            // type (1 byte) - clock
            view.setUint8(outOffset + 10, 0);
        }
        return 0;
    }
}
