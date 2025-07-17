local result = {}
local n = tonumber(ARGV[1])

for i = 1, n do
    local item = redis.call("LPOP", KEYS[1])
    if not item then
        break
    end
    table.insert(result, item)
end

return result