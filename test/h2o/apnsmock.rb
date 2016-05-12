LIMIT_APNS_TOKEN_BYTE_SIZE = 100

class ApnsMock
  def call(env)
    now = Time.now
    if /\/3\/device\/(.*)?$/.match(env["PATH_INFO"])
      Sleep::usleep( 750 + (rand(1500) - 750 ))
      token = $1
      if token.length > LIMIT_APNS_TOKEN_BYTE_SIZE || token == "baddevicetoken"
        print_time now
        return [400, {
          "content-type" => "application/json"
        }, ['{"reason":"BadDeviceToken"}']]
      elsif token == "unregistered"
        print_time now
        return [410, {
          "content-type" => "application/json"
        }, ['{"reason":"Unregistered","timestamp":1454402113}']]
      elsif token == "missingtopic"
        print_time now
        return [400, {
          "content-type" => "application/json"
        }, ['{"reason":"MissingTopic"}']]
      end
      print_time now
      return [200, {}, {}]
    else
      print_time now
      return [404, {
        "content-type" => "application/json"
      }, ['{"reason":"not found"}']]
    end
  end
end

def print_time(now)
  p Time.now - now
end

ApnsMock.new
