-- generate apn token function
local random = math.random
local function gen_apns_token()
    local template ='xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'
    return string.gsub(template, '[x]', function (c)
        math.randomseed(os.clock()*100000000000)
        local v = (c == 'x') and random(0, 0xf) or random(8, 0xb)
        return string.format('%x', v)
    end)
end

-- paylaod
payload = function()
    return '{"token":"'.. gen_apns_token() ..'", "payload": {"aps":{"alert":"hoge","badge":1,"sound":"default"},"mio":"hoge","uid":"hoge"}}'
end
payloads = "[" .. payload()

-- create bulk
for i = 2, 30 do
    payloads = payloads .. "," .. payload()
end
payloads = payloads .. "]"

wrk.method = "POST"
wrk.body   = payloads
wrk.port = 38103
wrk.path = "/push/apns"
wrk.headers["Content-Type"] = "application/json"
