
import string, urlparse, httplib, urllib, random
import datetime, threading, sys, time, collections

AllShortcodes = [
    ('http://10.130.32.52:58080/driver-container/incomingMessages/USSybase1/sms', [90001], ['Hello', 'Ciao', 'Hola', 'Salut']),
    ('http://10.130.32.52:58080/driver-container/incomingMessages/USSybase2/sms', [90002], ['Hello', 'Ciao', 'Hola', 'Salut']),
    ('http://10.130.32.52:58080/driver-container/incomingMessages/USSybase3/sms', [90003], ['Hello', 'Ciao', 'Hola', 'Salut']),
    ('http://10.130.32.52:58080/driver-container/incomingMessages/USSybase4/sms', [90004], ['Hello', 'Ciao', 'Hola', 'Salut']),
    ('http://10.130.32.52:58080/driver-container/incomingMessages/USSybase5/sms', [90005], ['Hello', 'Ciao', 'Hola', 'Salut']),
    ('http://10.130.32.52:58080/driver-container/incomingMessages/USSybase6/sms', [90006], ['Hello', 'Ciao', 'Hola', 'Salut']),
    ('http://10.130.32.52:58080/driver-container/incomingMessages/USSybase7/sms', [90007], ['Hello', 'Ciao', 'Hola', 'Salut']),
    ('http://10.130.32.52:58080/driver-container/incomingMessages/USSybase8/sms', [90008], ['Hello', 'Ciao', 'Hola', 'Salut']),
    ('http://10.130.32.52:58080/driver-container/incomingMessages/USSybase9/sms', [90009], ['Hello', 'Ciao', 'Hola', 'Salut'])
]
shortcodes = AllShortcodes[:4]

message = string.Template("""<?xml version="1.0" ?>
<SMS_MO>
<MSISDN>${msisdn}</MSISDN>
<ORIGINATING_ADDRESS>${shortcode}</ORIGINATING_ADDRESS>
<MESSAGE>${text}</MESSAGE>
<PARAMETERS>
  <OPERATORID>${operatorId}</OPERATORID>
  <ACCOUNTID>19823</ACCOUNTID>
  <MESSAGEID>${messageId}</MESSAGEID>
  <OPERATOR_INFORMATION>
    <OPERATOR_STANDARD>GSM</OPERATOR_STANDARD>
    <OPERATOR_CODE>
      <MCC>310</MCC>
      <MNC>20</MNC>
    </OPERATOR_CODE>
  </OPERATOR_INFORMATION>
  <DCS>7b</DCS>
  <CLASS>2</CLASS>
  <RECEIVED_SERVICENUMBER>83118</RECEIVED_SERVICENUMBER>
  <KEYWORD>VELTI</KEYWORD>
  <RECEIVEDTIME>
    <DATE>Thu, 17 Oct 2011</DATE>
    <TIME>16:43:14</TIME>
  </RECEIVEDTIME>
</PARAMETERS>
</SMS_MO>
""")


def RandomMessageId():
    return "SyBlaster" + str(int(time.time() * 1000)) + str(random.randint(1, 1000000))


def CreateNewMO(shortcodes, keywords):
    now = datetime.datetime.utcnow()
    params = {
        'msisdn': ('693%07d' % random.randint(1, 1000000)),
        'shortcode': random.choice(shortcodes),
        'text': random.choice(keywords),
        'messageId': RandomMessageId(),
        'operatorId': '383', # AT&T
        'date': now.date().strftime('%a, %d %b %Y'),
        'time': now.time().strftime('%H:%M:%S')
    }
    return message.substitute(params)


class QPSMonitor:
    def __init__(self):
        self.Clear()

    def Clear(self):
        self.hits_per_code = collections.Counter()
        self.total_hits = 0
        self.touched_hits = 0
        self.touched_time = datetime.datetime.today()
        self.history = collections.deque([], 30)
        self.history.append((self.total_hits, self.touched_time))

    def Note(self, status):
        self.total_hits += 1
        self.hits_per_code[status] += 1

    def Touch(self, present = True):
        hits_diff = self.total_hits - self.touched_hits
        now = datetime.datetime.today()
        elapsed_secs = (now - self.touched_time).total_seconds()
        if present:
            print 'From %s To %s' % (self.touched_time, now)
            print '\tHits %s Diff %d QPS %s' % (self.total_hits, hits_diff, hits_diff / elapsed_secs)
            print '\t%s' % (self.hits_per_code,)
        self.history.append((self.total_hits, now))
        self.touched_hits = self.total_hits
        self.touched_time = now

    def Avg(self):
        self.Touch(False)
        hdiff = (self.history[len(self.history) - 1][0] - self.history[0][0])
        tdiff = (self.history[len(self.history) - 1][1] - self.history[0][1]).total_seconds()
        print "Avg QPS in last %s sec: %s" % (tdiff, hdiff / tdiff)

    def History(self):
        prev = self.history[0]
        print prev[1].strftime('%H:%M:%S'), prev[0]
        for t in map(None, self.history)[1:]:
            print "%s\t%s\t%s\t%s" % (t[1].strftime('%H:%M:%S'), t[0], t[0] - prev[0], (t[0] - prev[0]) / (t[1] - prev[1]).total_seconds())
            prev = t

        

QPS = QPSMonitor()

class SybaseBlaster(threading.Thread):
    def __init__(self, sc):
        threading.Thread.__init__(self)
        self.keep_running = True
        self.shortcode = sc

    def run(self):
        u = urlparse.urlparse(self.shortcode[0])
        conn = httplib.HTTPConnection(u.hostname, u.port)
        conn.connect()
        headers = {
            'Connection': 'keep-alive',
            'Content-Type': 'application/x-www-form-urlencoded'
        }
        status = ''
        while self.keep_running:
            try:
                conn.request('POST', u.path, urllib.urlencode({'XmlMsg': CreateNewMO(self.shortcode[1], self.shortcode[2])}), headers)
                resp = conn.getresponse()
                body = resp.read()
                status = resp.status
            except httplib.HTTPException as e:
                status = str(e)
            QPS.Note(status)
        conn.close()


blasters = collections.deque()
                                             
def StartBlasters(n):
    for i in xrange(n):
        t = SybaseBlaster(random.choice(shortcodes))
        blasters.append(t)
        t.start()


def StopBlasters(n = None):
    if n is None:
        for t in blasters:
            t.keep_running = False
        blasters.clear()
        QPS.Clear()
    else:
        random.shuffle(blasters)
        for i in xrange(n):
            t = blasters.pop()
            t.keep_running = False


if __name__ == '__main__':
    random.seed()
