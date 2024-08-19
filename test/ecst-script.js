import http from 'k6/http';
import { sleep } from 'k6';
import { SharedArray } from 'k6/data';
import { vu } from 'k6/execution';
import {check, fail} from "k6";
import {Counter, Trend} from "k6/metrics";


const getTryoutSuccessCounter = new Counter('custom_get_tryout_success');
const createSubmissionsSuccessCounter = new Counter('custom_submission_success');
const createAnswerSuccessCounter = new Counter('custom_answer_success');
const createUpdatedSuccessCounter = new Counter('custom_updated_answer_success');

const getTryoutFailedCounter = new Counter('custom_get_tryout_failed');
const createSubmissionFailedCounter = new Counter('custom_create_submission_failed');
const createAnswerFailedCounter = new Counter('custom_create_answer_failed');
const createUpdatedFailedCounter = new Counter('custom_updated_answer_success');


const getTryoutWaiting = new Trend('custom_get_tryout_waiting');
const createSubmissionWaiting = new Trend('custom_create_submission_waiting');
const createAnswerWaiting = new Trend('custom_create_answer_waiting');
const createLeaderboardSuccessCounter = new Counter('custom_leaderboard_success');
const createLeaderboardFailedCounter = new Counter('custom_create_leaderboard_failed');


export const options = {

    scenarios: {
        constant_request_rate: {
            executor: 'constant-arrival-rate',
            rate: 500,
            timeUnit: '1s', // 1000 iterations per second, i.e. 1000 RPS
            duration: '60s',
            preAllocatedVUs: 2000, // how large the initial pool of VUs would be
            //maxVUs: 1000, // if the preAllocatedVUs are not enough, we can initialize more
        },
    },

    // vus: 2000,
    // iterations: 10000,
    // duration: '300s',
    thresholds: {
        'http_req_duration{submission:post}': [],
        'http_req_duration{tryout:get}': [],
        'http_req_duration{answer:post}': [],
        'http_req_duration{leaderboard:get}': [],
        // 'http_req_duration{updated_answer:post}': [],
    },
};

// const users = new SharedArray('some data name', function () {
//     return JSON.parse(open('./participants_token_100.json')).users;
// });

export default function () {


    // const before = new Date().getTime();
    // const T = 5;
    // console.log(`id: ${users[vu.idInTest - 1].id}`);
    // Step 1: Send GET request to get tryout details
    let tryoutResponse = http.get('http://54.251.92.13:5000/api/tryout/detail?id=01904abe-21fc-7e80-bf36-acfe5d96f27e', {tags: {tryout:`get`}});
    // getTryoutWaiting.add(tryoutResponse.timings.waiting);
    // console.log(tryoutData)
    // console.log("tryout response status")
    // console.log(tryoutResponse.status)
    const checkTryout = check(tryoutResponse, {
        'is status OK': (r) => r.status === 200,
    });

    if (!checkTryout) {
        getTryoutFailedCounter.add(1)
        console.log(`get tryout failed: ${tryoutResponse.body}`)
        fail('Failed to get tryout');
    }

    let tryoutData = tryoutResponse.json().data.question;
    getTryoutSuccessCounter.add(1)


    let submissionPayload = {
        // token: `${users[vu.idInTest - 1].token}`,
        token: "RPj9pF9dsO74",
        tryout_id: "01904abe-21fc-7e80-bf36-acfe5d96f27e"
    };
    let submissionResponse = http.post("http://54.251.92.13:3000/api/submission/create", JSON.stringify(submissionPayload), {
        headers: { 'Content-Type': 'application/json' },
        tags: {submission:`post`}
    });

    const checkSubmission = check(submissionResponse, {
        'is status OK': (r) => r.status === 200,
    });

    if (!checkSubmission) {
        createSubmissionFailedCounter.add(1)
        console.log(`post submission failed: ${submissionResponse.body}`)
        fail('Failed to post submission');
    } else {
        sleep(0.6)
        createSubmissionsSuccessCounter.add(1);
        let submissionId = submissionResponse.json().data.id;
        for (let i = 0; i < tryoutData.length; i++) {
            // sleep(1)
            let question = tryoutData[i];
            // console.log("quqestion choice ke i")
            // console.log(question.choice)
            let randomChoice = question.choice[Math.floor(Math.random() * question.choice.length)];
            // console.log(randomChoice)

            let answerPayload = {
                user_test_submission_id: submissionId,
                question_id: question.id,
                choice_id: randomChoice.id
            };
            // Your logic for submitting answerPayload goes here
            // For example, you might want to send this payload to a server
            // or process it in some other way
            // console.log(answerPayload); // Example: Output the payload to console
            let answer = http.post("http://54.251.92.13:3000/api/answer/create", JSON.stringify(answerPayload), {
                headers: { 'Content-Type': 'application/json' },
                tags: {answer:`post`}
            });
            // console.log("answer response status")
            // console.log(answer.status)
            // console.log(answer.data)
            // createAnswerWaiting.add(answer.timings.waiting);
            const checkAnswer = check(answer, {
                'is status OK': (r) => r.status === 200,
            });

            if (!checkAnswer) {
                createAnswerFailedCounter.add(1)
                console.log(`post answer failed: ${checkAnswer.body}`)
                fail('Failed to post answer');
            }
            createAnswerSuccessCounter.add(1)
            sleep(0.6)

            // let isErrorAnswer = check(answer, {
            //     'is 500': (r) => r.status >= 500,
            // });
            // if (isErrorAnswer) {
            //     createAnswerFailedCounter.add(1);
            //     fail('Internal error');
            // }
            // let isAnsmClientError = check(submissionResponse, {
            //     'is 400': (r) => r.status >= 400 && r.status < 500,
            // });
            // if (isAnsmClientError) {
            //     createAnswerFailedCounter.add(1);
            //     fail('client error');
            // }
            // console.log(`Step 3 : VU ID: ${__VU} ` + "- URL: " + answer.url + " - Status Code: " + answer.status);
        }

    }





    // let isErrorSubmission = check(submissionResponse, {
    //     'is 500': (r) => r.status >= 500,
    // });
    // if (isErrorSubmission) {
    //     createSubmissionFailedCounter.add(1);
    //     fail('Internal error');
    // }
    // let isSubmClientError = check(submissionResponse, {
    //     'is 400': (r) => r.status >= 400 && r.status < 500,
    // });
    // if (isSubmClientError) {
    //     createSubmissionFailedCounter.add(1);
    //     fail('client error');
    // }
    //


    // let updatedAnswerPayload = {
    //     submission_id: submissionId,
    // };
    // let answeredResponse = http.post("http://ip-172-31-23-213.ap-southeast-1.compute.internal:3000/api/submission/update", JSON.stringify(updatedAnswerPayload), {
    //     headers: { 'Content-Type': 'application/json' },
    //     tags: {updated_answer:`post`}
    // });
    //
    // if (answeredResponse.status === 200){
    //     createUpdatedSuccessCounter.add(1);
    // }
    //
    // let isErrorUpdated = check(answeredResponse, {
    //     'is 500 or higher': (r) => r.status >= 500,
    // });
    //
    // if (isErrorUpdated) {
    //     createUpdatedFailedCounter.add(1);
    //     fail('Internal error');
    // }
    // let isUpdatedClientError = check(answeredResponse, {
    //     'is 400': (r) => r.status >= 400 && r.status < 500,
    // });
    // if (isUpdatedClientError) {
    //     createUpdatedFailedCounter.add(1);
    //     fail('client error');
    // }


    let leaderboardResponse = http.get('http://54.251.92.13:3001/api/http/get?page=1&size=15&tid=01904abe-21fc-7e80-bf36-acfe5d96f27e', {tags: {leaderboard:`get`}});

    const checkLeaderboard = check(leaderboardResponse, {
        'is status OK': (r) => r.status === 200,
    });

    if (!checkLeaderboard) {
        createLeaderboardFailedCounter.add(1)
        console.log(`get leaderboard failed: ${leaderboardResponse.body}`)
        fail('Failed to get leaderboard');
    }else {
        createLeaderboardSuccessCounter.add(1)
    }
}

