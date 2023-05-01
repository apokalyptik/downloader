import {useState} from 'react';
import logo from './assets/images/logo-universal.png';
import './App.css';
import {InitDownload} from "../wailsjs/go/main/App";
import * as runtime from "../wailsjs/runtime";

function App() {

    const [resultText, setResultText] = useState("");
    const [errorText, setErrorText] = useState("");
    const [url, setUrl] = useState('');
    const [downloading, setDownloading] = useState("");

    const updateUrl = (e) => setUrl(e.target.value);
    const updateResultText = (result) => setResultText(result);
    const updateErrorText = (result) => setErrorText(result);

    function download() {
        setErrorText("")
        setDownloading(true)
        InitDownload(url)
            .catch(updateErrorText);
    }

    runtime.EventsOn("progress", updateResultText)
    runtime.EventsOn("done", function() {
        setDownloading(false)
    })

    return (
        <div id="App">
            <div id="result" className="result">{resultText}</div>
            <div id="error" className="error">{errorText}</div>
            <div id="input" className="input-box">
                <input id="url" className="input" onChange={updateUrl} autoComplete="off" name="input" type="text" placeholder="Please enter the URL you wish to download"/>
                <button id="download" disabled={downloading} className="btn" onClick={download}>Download</button>
            </div>
            <div style={{width:"70%",marginLeft:"15%",textAlign:"left"}}>
                <ol>
                    <li>Paste, or enter, the URL you wish to download above.</li>
                    <li>When you click "Download" you will be asked to choose a filename to save as.</li>
                    <li>If the file exists you will be asked if you want to replace it.</li>
                    <li>Choose to replace it and the download will resume from where it left off.</li>
                    <li>The download will keep retrying, indefinitely, until it is able to finish downloading.</li>
                </ol>
            </div>
        </div>
    )
}

export default App
