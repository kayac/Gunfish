-- generates payload
local random = math.random
local function gen_apns_token()
    local template ='xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx'
    return string.gsub(template, '[x]', function (c)
        math.randomseed(os.clock()*100000000000)
        local v = (c == 'x') and random(0, 0xf) or random(8, 0xb)
        return string.format('%x', v)
    end)
end

local function gen_error_token()
    math.randomseed(os.clock()*100000000000)
    local err_type =random(1,3)
    if err_type == 1 then
        return "missingtopic"
    elseif err_type == 2 then
        return "baddevicetoken"
    else
        return "unregistered"
    end
end

-- paylaod
payload = function(is_error)
    if is_error == 0 then
        token = gen_apns_token()
    else
        token = gen_error_token()
    end
    return '{"token":"'.. token ..'", "payload": {"aps":{"alert":"hoge","badge":1,"sound":"default"},"mio":"hoge","uid":"hoge"}}'
end

-- create bulk. 50:1 = success:error
payloads = "[" .. payload(0)
for i = 2, 200 do
    if i % 50 == 0 then
        payloads = payloads .. "," .. payload(1)
    else
        payloads = payloads .. "," .. payload(0)
    end
end
payloads = payloads .. "]"

-- POST
wrk.method = "POST"
wrk.body   = payloads
wrk.port = 38103
wrk.path = "/push/apns"
wrk.headers["Content-Type"] = "application/json"
