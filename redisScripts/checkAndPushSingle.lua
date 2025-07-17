if redis.call("SADD", KEYS[1], ARGV[1]) == 1 then
    return redis.call("RPUSH", KEYS[2], ARGV[1])
else
    return 0
end