import {useState, useEffect} from 'react';
import logo from './assets/images/logo-universal.png';
import './App.css';
import {Add, FrontEndReady, Delete, Pause, Save} from "../wailsjs/go/main/DownloaderBackend";
import * as runtime from "../wailsjs/runtime";

function humanizeSize(bytes) {
    if ( bytes < 1024 ) {
        return "" + bytes + "b"
    }
    bytes = bytes/1024
    if ( bytes < 1024 ) {
        return "" + Math.round(bytes*100)/100 + "KB"
    }
    bytes = bytes/1024
    if ( bytes < 1024 ) {
        return "" + Math.round(bytes*100)/100 + "MB"
    }
    bytes = bytes/1024
    if ( bytes < 1024 ) {
        return "" + Math.round(bytes*100)/100 + "GB"
    }
    bytes = bytes/1024
    if ( bytes < 1024 ) {
        return "" + Math.round(bytes*100)/100 + "TB"
    }
}

function renderDownloads(downloads) {
    return downloads.map(function(ele, idx) {
        var filenameGuess = ele.FilenameGuessFromURL
        if ( ele.FilenameGuessFromHEAD != "" ) {
            filenameGuess = ele.FilenameGuessFromHEAD
        }
        var save = null
        var pause = null
        var speed = null
        var resumable = null
        if ( ele.Complete ) {
            save = ( <div className="saveButton" onClick={function() { Save( ele.URL, false ); }}>Save</div> )
            speed = ( <span>Download has completed</span> )
        } else {
            if ( ele.Resumable ) {
                resumable = "resumeable:✓ "
            } else {
                resumable = "resumeable:✗ "
            }
            speed = ( <span>@ {humanizeSize(ele.BytesPerSecond)}/s</span> )
            if ( ele.Paused ) {
                    pause = ( <div className="pauseButton" onClick={function() { Pause( ele.URL, false ); }}>⏵</div> )
            } else {
                    pause = ( <div className="pauseButton" onClick={function() { Pause( ele.URL, true ); }}>⏸</div> )
            }
        }
        return (
            <div key={idx} className="downloadSummary">
                <div className="cancelButton" onClick={function() { Delete( ele.URL ); }}>x</div>
                {pause}
                {save}
                <h3 className="filename">{filenameGuess}</h3>
                <div className="url">{ele.URL}</div>
                <div className="bytes">{resumable} {humanizeSize(ele.DownloadedBytes)} of {humanizeSize(ele.Bytes)} on try #{ele.AttemptCounter} {speed}</div>
                <div className="progress"><div className="pct" style={{width: ele.Pct + "%"}}></div></div>
            </div>
        )
    })
    return JSON.stringify(downloads)
}

function App() {

    const [url, setUrl] = useState('');
    const updateUrl = (e) => setUrl(e.target.value);
    const [viewing, setViewing] = useState(null)
    const [downloads, setDownloads] = useState([]);

    useEffect(
        function() {
            // This only happens ONCE. It happens AFTER the initial render.
            FrontEndReady();
        },
        []
    )

    function add() {
        Add(url)
    }

    runtime.EventsOn("updateDownloads", setDownloads)
    runtime.EventsOn("setUrl", setUrl);

    return (
        <div id="App">
            <div id="input" className="input-box">
                <input id="url" value={url} className="input" onChange={updateUrl} autoComplete="off" name="input" type="text" placeholder="Please enter the URL you wish to download"/>
                <button id="download" className="btn" onClick={add}>Download</button>
            </div>
            <div id="downloadsection">
                <div id="downloadlist">{renderDownloads(downloads)}</div>
            </div>
        </div>
    )
}

export default App
