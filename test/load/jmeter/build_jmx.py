#!/usr/bin/env python3
"""Генерує тестовий план JMeter (ЛР9, завдання 2): python3 build_jmx.py > roomflow.jmx

Профіль: 10 користувачів з першої секунди + ще 90, що додаються протягом RAMP секунд (разом 10 -> 100).
Параметри (-J): host, port, endpoint (health|availability|booking), ramp, duration.
"""
from xml.sax.saxutils import escape

P = lambda name, default: f"${{__P({name},{default})}}"


def prop(kind, name, value):
    return f'<{kind}Prop name="{name}">{escape(str(value))}</{kind}Prop>'


def http_sampler(name, method, path, body=None):
    args = ""
    if body:
        args = ('<boolProp name="HTTPSampler.postBodyRaw">true</boolProp>'
                '<elementProp name="HTTPsampler.Arguments" elementType="Arguments"><collectionProp name="Arguments.arguments">'
                '<elementProp name="" elementType="HTTPArgument"><boolProp name="HTTPArgument.always_encode">false</boolProp>'
                f'{prop("string", "Argument.value", body)}{prop("string", "Argument.metadata", "=")}</elementProp>'
                '</collectionProp></elementProp>')
    else:
        args = '<elementProp name="HTTPsampler.Arguments" elementType="Arguments"><collectionProp name="Arguments.arguments"/></elementProp>'
    return (f'<HTTPSamplerProxy guiclass="HttpTestSampleGui" testclass="HTTPSamplerProxy" testname="{name}" enabled="true">'
            f'{args}{prop("string", "HTTPSampler.domain", P("host", "localhost"))}{prop("string", "HTTPSampler.port", P("port", "8080"))}'
            f'{prop("string", "HTTPSampler.protocol", "http")}{prop("string", "HTTPSampler.path", path)}{prop("string", "HTTPSampler.method", method)}'
            f'{prop("bool", "HTTPSampler.follow_redirects", "true")}{prop("bool", "HTTPSampler.use_keepalive", "true")}'
            '</HTTPSamplerProxy><hashTree>'
            '<ResponseAssertion guiclass="AssertionGui" testclass="ResponseAssertion" testname="2xx or 409" enabled="true">'
            '<collectionProp name="Asserion.test_strings"><stringProp name="1">(200|201|409)</stringProp></collectionProp>'
            '<stringProp name="Assertion.custom_message"></stringProp><stringProp name="Assertion.test_field">Assertion.response_code</stringProp>'
            '<boolProp name="Assertion.assume_success">true</boolProp><intProp name="Assertion.test_type">1</intProp></ResponseAssertion><hashTree/>'
            '</hashTree>')


def if_controller(condition, inner):
    return (f'<IfController guiclass="IfControllerPanel" testclass="IfController" testname="if {escape(condition, {chr(34): '&quot;'})}" enabled="true">'
            f'{prop("string", "IfController.condition", condition)}{prop("bool", "IfController.evaluateAll", "false")}'
            f'{prop("bool", "IfController.useExpression", "true")}</IfController><hashTree>{inner}</hashTree>')


day = "${__javaScript(new Date(Date.UTC(2027,0,1)+(%s)*86400000).toISOString().slice(0\\,10))}"
tenant = "hotel-${__Random(0,19)}"
room = "r${__Random(1,10)}"
samplers = (
    if_controller('${__jexl3("${__P(endpoint,availability)}" == "health")}', http_sampler("health", "GET", "/healthz"))
    + if_controller('${__jexl3("${__P(endpoint,availability)}" == "availability")}',
                    http_sampler("availability", "GET", f"/api/rooms/{room}/availability?from=2027-01-01&to=2027-03-02"))
    + if_controller('${__jexl3("${__P(endpoint,availability)}" == "booking")}',
                    http_sampler("booking", "POST", "/api/bookings",
                                 '{"room_id":"%s","guest_name":"jmeter","guest_email":"jm@example.com","guests":2,"check_in":"%s","check_out":"%s"}'
                                 % (room, day % "${__Random(30000,60000,d)}", day % "${d}+1")))
)
headers = ('<HeaderManager guiclass="HeaderPanel" testclass="HeaderManager" testname="headers" enabled="true"><collectionProp name="HeaderManager.headers">'
           f'<elementProp name="" elementType="Header">{prop("string", "Header.name", "X-Tenant-ID")}{prop("string", "Header.value", tenant)}</elementProp>'
           f'<elementProp name="" elementType="Header">{prop("string", "Header.name", "Content-Type")}{prop("string", "Header.value", "application/json")}</elementProp>'
           '</collectionProp></HeaderManager><hashTree/>')


def thread_group(name, threads, ramp):
    return (f'<ThreadGroup guiclass="ThreadGroupGui" testclass="ThreadGroup" testname="{name}" enabled="true">'
            f'{prop("string", "ThreadGroup.on_sample_error", "continue")}'
            '<elementProp name="ThreadGroup.main_controller" elementType="LoopController" guiclass="LoopControlPanel" testclass="LoopController">'
            f'{prop("bool", "LoopController.continue_forever", "false")}{prop("int", "LoopController.loops", "-1")}</elementProp>'
            f'{prop("string", "ThreadGroup.num_threads", threads)}{prop("string", "ThreadGroup.ramp_time", ramp)}'
            f'{prop("bool", "ThreadGroup.scheduler", "true")}{prop("string", "ThreadGroup.duration", P("duration", "90"))}'
            f'{prop("string", "ThreadGroup.delay", "0")}{prop("bool", "ThreadGroup.same_user_on_next_iteration", "true")}'
            f'</ThreadGroup><hashTree>{headers}{samplers}</hashTree>')


print('<?xml version="1.0" encoding="UTF-8"?>\n<jmeterTestPlan version="1.2" properties="5.0" jmeter="5.5"><hashTree>'
      '<TestPlan guiclass="TestPlanGui" testclass="TestPlan" testname="RoomFlow load test" enabled="true">'
      '<elementProp name="TestPlan.user_defined_variables" elementType="Arguments"><collectionProp name="Arguments.arguments"/></elementProp>'
      '<boolProp name="TestPlan.functional_mode">false</boolProp><boolProp name="TestPlan.serialize_threadgroups">false</boolProp></TestPlan><hashTree>'
      + thread_group("base users (10)", "10", "1")
      + thread_group("ramp users (+90)", "90", P("ramp", "60"))
      + '</hashTree></hashTree></jmeterTestPlan>')
