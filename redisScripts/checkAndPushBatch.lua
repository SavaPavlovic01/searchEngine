local added = 0

for i, url in ipairs(ARGV) do
    if redis.call("SADD", KEYS[1], url) == 1 then
        redis.call("RPUSH", KEYS[2], url)
        added += 1
    end
end

return added  