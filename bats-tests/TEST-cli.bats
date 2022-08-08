#!/usr/bin/env bats

@test "test-shebang.lua" {
    run "${BATS_TEST_DIRNAME}/testdata/test-shebang.lua"
    echo "$output"
    echo "$status"
    ((!status))
    [[ "${lines[0]}" == 'hello world' ]]
    [[ "${lines[1]}" == "${BATS_TEST_DIRNAME}/testdata/test-shebang.lua" ]]
    [[ "${lines[2]}" == '[]' ]]
}

@test "test-shebang.lua --flagarg argvalue" {
    run "${BATS_TEST_DIRNAME}/testdata/test-shebang.lua" --flagarg argvalue
    echo "$output"
    echo "$status"
    ((!status))
    [[ "${lines[0]}" == 'hello world' ]]
    [[ "${lines[1]}" == "${BATS_TEST_DIRNAME}/testdata/test-shebang.lua" ]]
    [[ "${lines[2]}" == '["--flagarg","argvalue"]' ]]
}

@test "test-shebang-argparse.lua" {
    run "${BATS_TEST_DIRNAME}/testdata/test-shebang-argparse.lua"
    echo "$output"
    echo "$status"
    ((!status))
    expected="$(jq -nSc '{default: "default", rest:[]}')"
    echo "$expected"
    [[ "$output" == "$expected" ]]
}

@test "test-shebang-argparse.lua -t" {
    run "${BATS_TEST_DIRNAME}/testdata/test-shebang-argparse.lua" -t
    echo "$output"
    echo "$status"
    ((!status))
    expected="$(jq -nSc '{default: "default", rest: [], test: true}')"
    echo "$expected"
    [[ "$output" == "$expected" ]]
}

@test "test-shebang-argparse.lua --test" {
    run "${BATS_TEST_DIRNAME}/testdata/test-shebang-argparse.lua" --test
    echo "$output"
    echo "$status"
    ((!status))
    expected="$(jq -nSc '{default: "default", rest: [], test: true}')"
    echo "$expected"
    [[ "$output" == "$expected" ]]
}

@test "test-shebang-argparse.lua -c FOO" {
    run "${BATS_TEST_DIRNAME}/testdata/test-shebang-argparse.lua" -c FOO
    echo "$output"
    echo "$status"
    ((!status))
    expected="$(jq -nSc '{choice: "FOO", default: "default", rest: []}')"
    echo "$expected"
    [[ "$output" == "$expected" ]]
}

@test "test-shebang-argparse.lua -c BAR" {
    run "${BATS_TEST_DIRNAME}/testdata/test-shebang-argparse.lua" -c BAR
    echo "$output"
    echo "$status"
    ((!status))
    expected="$(jq -nSc '{choice: "BAR", default: "default", rest: []}')"
    echo "$expected"
    [[ "$output" == "$expected" ]]
}

@test "test-shebang-argparse.lua -c BAZ" {
    run "${BATS_TEST_DIRNAME}/testdata/test-shebang-argparse.lua" -c BAZ
    echo "$output"
    echo "$status"
    ((!status))
    expected="$(jq -nSc '{choice: "BAZ", default: "default", rest: []}')"
    echo "$expected"
    [[ "$output" == "$expected" ]]
}

@test "test-shebang-argparse.lua -c BAD_CHOICE fails" {
    run "${BATS_TEST_DIRNAME}/testdata/test-shebang-argparse.lua" -c BAD_CHOICE
    echo "$output"
    echo "$status"
    ((status))
    [[ "$output" =~ ^Usage: ]]
}
