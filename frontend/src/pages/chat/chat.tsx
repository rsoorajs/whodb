/*
 * Copyright 2025 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import classNames from "classnames";
import { AnimatePresence, motion } from "framer-motion";
import { map } from "lodash";
import { cloneElement, FC, KeyboardEventHandler, MouseEvent, ReactElement, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { v4 } from "uuid";
import { ActionButton, AnimatedButton, Button } from "../../components/button";
import { createDropdownItem, Dropdown, DropdownWithLabel, IDropdownItem } from "../../components/dropdown";
import { CodeEditor } from "../../components/editor";
import { Icons } from "../../components/icons";
import { Input, InputWithlabel } from "../../components/input";
import { Loading } from "../../components/loading";
import { InternalPage } from "../../components/page";
import { Table } from "../../components/table";
import { InternalRoutes } from "../../config/routes";
import { AiChatMessage, GetAiChatQuery, useGetAiChatLazyQuery, useGetAiModelsLazyQuery, useGetAiProvidersLazyQuery } from '@graphql';
import { availableExternalModelTypes, AIModelsActions } from "../../store/ai-models";
import { ensureModelTypesArray, ensureModelsArray } from "../../utils/ai-models-helper";
import { notify } from "../../store/function";
import { useAppDispatch, useAppSelector } from "../../store/hooks";
import { chooseRandomItems } from "../../utils/functions";
import { chatExamples } from "./examples";
const logoImage = "/images/logo.png";
import { HoudiniActions } from "../../store/chat";
import { reduxStore } from "../../store";
import { loadEEComponent, isEEFeatureEnabled } from "../../utils/ee-loader";
import { isEEMode } from "@/config/ee-imports";

// Lazy load chart components if EE is enabled
const LineChart = isEEFeatureEnabled('dataVisualization') ? loadEEComponent(
    () => import('@ee/components/charts/line-chart').then(m => ({ default: m.LineChart })),
    () => null
) : () => null;

const PieChart = isEEFeatureEnabled('dataVisualization') ? loadEEComponent(
    () => import('@ee/components/charts/pie-chart').then(m => ({ default: m.PieChart })),
    () => null
) : () => null;

const thinkingPhrases = [
    "Thinking",
    "Pondering life’s mysteries",
    "Consulting the cloud oracles",
    "Googling furiously (just kidding)",
    "Aligning the neural networks",
    "Making it up as I go (shh)",
    "Counting virtual sheep",
    "Channeling Einstein",
    "Tuning my algorithms",
    "Drinking a byte of coffee",
    "Running in circles virtually.",
    "Pretending to be busy",
    "Loading witty comeback",
    "Downloading some wisdom",
    "Cooking up some data stew",
    "Doing AI things™",
    "Hacking the mainframe (for fun)",
    "Staring into the digital abyss",
    "Flipping a quantum coin",
    "Reading your mind (ethically)",
    "Sharpening my sarcasm",
    "Checking my vibes",
    "Simulating deep thought",
    "Rewiring my circuits",
    "Polishing my crystal processor"
  ];
  

type TableData = GetAiChatQuery["AIChat"][0]["Result"];

const TablePreview: FC<{ type: string, data: TableData, text: string }> = ({ type, data, text }) => {
    const [showSQL, setShowSQL] = useState(false);

    const handleCodeToggle = useCallback(() => {
        setShowSQL(status => !status);
    }, []);

    return <div className="flex flex-col w-[calc(100%-50px)] group/table-preview gap-2 relative">
        <div className="absolute -top-3 -left-3 opacity-0 group-hover/table-preview:opacity-100 transition-all z-[1]">
            <ActionButton containerClassName="w-8 h-8" className="w-5 h-5" icon={cloneElement(showSQL ? Icons.Tables : Icons.Code, {
                className: "w-6 h-6 stroke-white",
            })} onClick={handleCodeToggle} />
        </div>
        <div className="flex items-center gap-4 overflow-hidden break-all leading-6 shrink-0 h-full w-full">
            {
                showSQL
                ? <div className="h-[150px] w-full">
                    <CodeEditor value={text} />
                </div>
                :  (data != null && data.Rows.length > 0) || type === "sql:get"
                    ? <Table className="h-[250px]" columns={data?.Columns.map(c => c.Name) ?? []} columnTags={data?.Columns.map(c => c.Type)}
                        rows={data?.Rows ?? []} totalPages={1} currentPage={1} disableEdit={true} />
                    : <div className="bg-white/10 text-neutral-800 dark:text-neutral-300 rounded-lg p-2 flex gap-2">
                        Action Executed ({type.toUpperCase().split(":")?.[1]})
                        {Icons.CheckCircle}
                    </div>
            }
        </div>
    </div>
}

type IChatMessage = AiChatMessage & {
    isUserInput?: boolean;
};


export const externalModelTypes = map(availableExternalModelTypes, (model) => createDropdownItem(model, (Icons.Logos as Record<string, ReactElement>)[model]));

export const ChatPage: FC = () => {
    const [query, setQuery] = useState("");
    const [addExternalModel, setAddExternalModel] = useState(false);
    const [externalModelType, setExternalModel] = useState(externalModelTypes[0]);
    const [externalModelToken, setExternalModelToken] = useState<string>();
    const modelType = useAppSelector(state => state.aiModels.current);
    const modelTypesRaw = useAppSelector(state => state.aiModels.modelTypes);
    const modelTypes = ensureModelTypesArray(modelTypesRaw);
    const currentModel = useAppSelector(state => state.aiModels.currentModel);
    const modelsRaw = useAppSelector(state => state.aiModels.models);
    const models = ensureModelsArray(modelsRaw);
    const chats = useAppSelector(state => state.houdini.chats);
    const [modelAvailable, setModelAvailable] = useState(true);
    const [getAiProviders, ] = useGetAiProvidersLazyQuery();
    const dispatch = useAppDispatch();
    const [getAIModels, { loading: getAIModelsLoading }] = useGetAiModelsLazyQuery({
        onError() {
            setModelAvailable(false);
            dispatch(AIModelsActions.setModels([]));
            dispatch(AIModelsActions.setCurrentModel(undefined));
        },
        fetchPolicy: "network-only",
    });
    const [getAIChat, { loading: getAIChatLoading }] = useGetAiChatLazyQuery();
    const scrollContainerRef = useRef<HTMLDivElement>(null);
    const schema = useAppSelector(state => state.database.schema);
    const [currentSearchIndex, setCurrentSearchIndex] = useState<number>();

    const loading = useMemo(() => {
        return getAIChatLoading || getAIModelsLoading;
    }, [getAIModelsLoading, getAIChatLoading]);

    const handleAIModelTypeChange = useCallback((item: IDropdownItem) => {
        setModelAvailable(true);
        getAIModels({
            variables: {
                providerId: item.id,
                modelType: item.label,
                token: item.extra?.token,
            },
            onCompleted(data) {
                dispatch(AIModelsActions.setModels(data.AIModel));
                if (data.AIModel.length > 0) {
                    dispatch(AIModelsActions.setCurrentModel(data.AIModel[0]));
                }
            },
        });
    }, [dispatch, getAIModels]);

    const handleAIModelChange = useCallback((item: IDropdownItem) => {
        dispatch(AIModelsActions.setCurrentModel(item.id));
    }, [dispatch]);

    const handleAIModelRemove = useCallback((_: MouseEvent<HTMLDivElement>, item?: IDropdownItem) => {
        if (modelType?.id === item!.id) {
            dispatch(AIModelsActions.setModels([]));
            dispatch(AIModelsActions.setCurrentModel(undefined));
        }
        dispatch(AIModelsActions.removeAIModelType({ id: item!.id }));
    }, [dispatch, modelType?.id]);

    const examples = useMemo(() => {
        return chooseRandomItems(chatExamples);
    }, []);

    const handleSubmitQuery = useCallback(() => {
        const sanitizedQuery = query.trim();
        if (modelType == null || sanitizedQuery.length === 0) {
            return;
        }

        dispatch(HoudiniActions.addChatMessage({ Type: "message", Text: sanitizedQuery, isUserInput: true, }));
        setTimeout(() => {
            if (scrollContainerRef.current != null) {
                scrollContainerRef.current.scroll({
                    top: scrollContainerRef.current.scrollHeight,
                    behavior: "smooth",
                });
            }
        }, 250);
        getAIChat({
            variables: {
                modelType: modelType.modelType,
                token: modelType.token,
                query: sanitizedQuery,
                model: currentModel ?? "",
                previousConversation: chats.map(chat => `${chat.isUserInput ? "<User>" : "<System>"}${chat.Text}${chat.isUserInput ? "</User>" : "</System>"}`).join("\n"),
                schema,
            },
            onCompleted(data) {
                const systemChats: IChatMessage[] = data.AIChat.map(chat => {
                    if (chat.Type.startsWith("sql")) {
                        return {
                            Type: chat.Type,
                            Text: chat.Text,
                            Result: chat.Result as AiChatMessage["Result"],
                        }
                    }
                    return {
                        Type: chat.Type,
                        Text: chat.Text,
                    }
                });
                for (const systemChat of systemChats) {
                    dispatch(HoudiniActions.addChatMessage(systemChat));
                }
                setTimeout(() => {
                    if (scrollContainerRef.current != null) {
                        scrollContainerRef.current.scroll({
                            top: scrollContainerRef.current.scrollHeight,
                            behavior: "smooth",
                        });
                    }
                }, 250);
            },
            onError(error) {
                notify("Unable to query. Try again. "+error.message, "error");
            },
        });
        setQuery("");
    }, [chats, currentModel, getAIChat, modelType, query, schema]);

    const handleKeyUp: KeyboardEventHandler<HTMLInputElement> = useCallback((e) => {
        if (e.key === "ArrowUp") {
          const foundSearchIndex = currentSearchIndex != null ? currentSearchIndex - 1 : chats.length - 1;
          let searchIndex = foundSearchIndex;

          while (searchIndex >= 0) {
            if (chats[searchIndex].isUserInput) {
              setCurrentSearchIndex(searchIndex);
              setQuery(chats[searchIndex].Text);
              return;
            }
            searchIndex--;
          }

          if (currentSearchIndex !== chats.length - 1) {
            searchIndex = chats.length - 1;
            while (searchIndex > foundSearchIndex) {
              if (chats[searchIndex].isUserInput) {
                setCurrentSearchIndex(searchIndex);
                setQuery(chats[searchIndex].Text);
                return;
              }
              searchIndex--;
            }
          }
        }
    }, [chats, currentSearchIndex]);

    const handleSelectExample = useCallback((example: string) => {
        setQuery(example);
    }, []);

    const handleAddExternalModel = useCallback(() => {
        setAddExternalModel(status => !status);
    }, []);

    const handleExternalModelChange = useCallback((item: IDropdownItem) => {
        setExternalModel(item);
    }, []);

    const handleExternalModelSubmit = useCallback(() => {
        dispatch(AIModelsActions.setCurrentModel(undefined));
        dispatch(AIModelsActions.setModels([]));
        getAIModels({
            variables: {
                modelType: externalModelType.id,
                token: externalModelToken,
            },
            onCompleted(data) {
                dispatch(AIModelsActions.setModels(data.AIModel));
                const id = v4();
                dispatch(AIModelsActions.addAIModelType({
                    id,
                    modelType: externalModelType.id,
                    token: externalModelToken,
                }));
                dispatch(AIModelsActions.setCurrentModelType({ id }));
                setExternalModel(externalModelTypes[0]);
                setExternalModelToken("");
                setAddExternalModel(false);
                if (data.AIModel.length > 0) {
                    dispatch(AIModelsActions.setCurrentModel(data.AIModel[0]));
                }
            },
            onError(error) {
                notify(`Unable to connect to the model: ${error.message}`, "error");
            },
        });
    }, [getAIModels, externalModelType.id, externalModelToken, dispatch]);

    const handleOpenDocs = useCallback(() => {
        window.open("https://whodb.com/docs/usage-houdini/what-is-houdini", "_blank");
    }, []);

    useEffect(() => {
        getAiProviders({
            onCompleted(data) {
                const aiProviders = data.AIProviders || [];
                const modelTypesState = ensureModelTypesArray(reduxStore.getState().aiModels.modelTypes);
                const initialModelTypes = modelTypesState.filter(model => {
                const existingModel = aiProviders.find(provider => provider.ProviderId === model.id);
                return existingModel != null || (model.token != null && model.token !== "");
                });

                // Filter out providers that already exist in modelTypes
                const newProviders = aiProviders.filter(provider =>
                !initialModelTypes.some(model => model.id === provider.ProviderId)
                );

                const finalModelTypes = [
                ...newProviders.map(provider => ({
                    id: provider.ProviderId,
                    modelType: provider.Type,
                    isEnvironmentDefined: provider.IsEnvironmentDefined,
                })),
                ...initialModelTypes
                ];

                // Check if current model type exists in final model types
                const currentModelType = reduxStore.getState().aiModels.current;
                if (currentModelType && !finalModelTypes.some(model => model.id === currentModelType.id)) {
                dispatch(AIModelsActions.setCurrentModelType({ id: "" }));
                dispatch(AIModelsActions.setModels([]));
                dispatch(AIModelsActions.setCurrentModel(undefined));
                }

                dispatch(AIModelsActions.setModelTypes(finalModelTypes));
                getAIModels({
                    variables: {
                        providerId: currentModelType?.id,
                        modelType: currentModelType?.modelType ?? "",
                        token: currentModelType?.token ?? "",
                    },
                });
            },
        });

        const modelType = modelTypes[0];
        if (modelType == null || models.length > 0) {
            return;
        }
        handleAIModelTypeChange({
            id: modelType.id,
            label: modelType.modelType,
            extra: {
                token: modelType.token,
            },
        });
    // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    const modelTypesDropdownItems = useMemo(() => {
        return modelTypes.filter(modelType => modelType != null && modelType.modelType != null).map(modelType => ({
            id: modelType.id,
            label: modelType.modelType,
            icon: (Icons.Logos as Record<string, ReactElement>)[modelType.modelType.replace("-", "")],
            extra: {
                token: modelType.token,
            }
        }));
    }, [modelTypes]);

    const disableAll = useMemo(() => {
        return models.length === 0 || !modelAvailable;
    }, [modelAvailable, models.length]);

    const disableChat = useMemo(() => {
        return loading || models.length === 0 || !modelAvailable;
    }, [loading, modelAvailable, models.length]);

    const handleClear = useCallback(() => {
        dispatch(HoudiniActions.clear());
    }, []);

    const handleAIProviderChange = useCallback((item: IDropdownItem) => {
        dispatch(AIModelsActions.setCurrentModelType({ id: item.id }));
        handleAIModelTypeChange(item);
    }, [handleAIModelTypeChange]);

    const modelDropdownItems = useMemo(() => {
        return models.map(model => createDropdownItem(model));
    }, [models]);

    return (
        <InternalPage routes={[InternalRoutes.Chat]}>
            <AnimatePresence mode="wait">
                {
                    addExternalModel &&
                    <div className="absolute inset-0 flex justify-center items-center">
                        <motion.div className="w-[min(450px,calc(100vw-20px))] shadow-2xl z-10 rounded-xl px-8 py-12 flex flex-col gap-2 relative overflow-hidden"
                            initial={{ y: 50, opacity: 0 }}
                            animate={{ y: 0, opacity: 1 }}
                            exit={{ y: 50, opacity: 0 }}>
                            <div className="absolute inset-0 bg-white/5 backdrop-blur-xl -z-10" />
                            <div className="text-neutral-800 dark:text-neutral-300 self-center flex gap-2 items-center">
                                Run Ollama locally <Button icon={Icons.RightArrowUp} label="Docs" onClick={handleOpenDocs} />
                            </div>
                            <div className="text-neutral-800 dark:text-neutral-300 my-4 w-full flex items-center gap-2">
                                <div className="border-t-[1px] border-t-white/5 rounded-lg grow border-dashed" /> or <div className="border-t-[1px] border-t-white/5 rounded-lg grow border-dashed" />
                            </div>
                            <div className="text-neutral-800 dark:text-neutral-300 self-center">
                                Add External Model
                            </div>
                            <DropdownWithLabel label="Model Type" items={externalModelTypes} fullWidth={true} value={externalModelType} onChange={handleExternalModelChange} />
                            <InputWithlabel label="Token" value={externalModelToken ?? ""} setValue={setExternalModelToken} type="password" />
                            <div className="flex items-center justify-between">
                                <AnimatedButton icon={Icons.CheckCircle} label="Cancel" onClick={handleAddExternalModel} />
                                <AnimatedButton icon={Icons.CheckCircle} label="Submit" onClick={handleExternalModelSubmit} disabled={getAIModelsLoading} />
                            </div>
                        </motion.div>
                    </div>
                }
            </AnimatePresence>
            <div className="flex flex-col justify-center items-center w-full h-full gap-2">
                <div className="flex w-full justify-between">
                    <div className={classNames("flex gap-2", {
                        "opacity-50 pointer-events-none": addExternalModel,
                    })}>
                        <Dropdown className="w-[200px]" value={modelType && {
                                id: modelType.id,
                                label: modelType.modelType || "",
                                icon: modelType.modelType ? (Icons.Logos as Record<string, ReactElement>)[modelType.modelType.replace("-", "")] : undefined,
                            }}
                            dropdownContainerHeight="max-h-[400px]"
                            items={modelTypesDropdownItems}
                            onChange={handleAIProviderChange}
                            action={<div onClick={handleAIModelRemove}>{cloneElement(Icons.Delete, {
                                className: "w-6 h-6 stroke-red-500"
                            })}</div>}
                            defaultItem={{ label: "Add External Model", icon: Icons.Add }}
                            onDefaultItemClick={handleAddExternalModel}
                            enableAction={(index) => {
                                const modelType = modelTypes.at(index);
                                return modelType?.token != null && !modelType?.isEnvironmentDefined;
                            }} />
                        {
                            modelAvailable
                            ? <Dropdown className="w-[200px]" value={currentModel ? createDropdownItem(currentModel) : undefined}
                                    items={modelDropdownItems} dropdownContainerHeight="max-h-[400px]"
                                    onChange={handleAIModelChange}
                                    loading={getAIModelsLoading} />
                            : <div className="text-neutral-500 w-[200px] rounded-lg bg-white/5 flex items-center pl-4">
                                Unavailable
                            </div>
                        }
                    </div>
                    <div className="flex gap-2">
                        <AnimatedButton label="New Chat" icon={Icons.Refresh} onClick={handleClear} disabled={loading} />
                    </div>
                </div>
                <div className={classNames("flex grow w-full rounded-xl overflow-hidden", {
                    "opacity-[4%] pointer-events-none": disableAll,
                })}>
                    {
                        chats.length === 0
                        ? <div className="flex flex-col justify-center items-center w-full gap-8">
                            <img src={logoImage} alt="clidey logo" className="w-auto h-16" />
                            <div className="flex flex-wrap justify-center items-center gap-4">
                                {
                                    examples.map((example, i) => (
                                        <div key={`chat-${i}`} className="flex flex-col gap-2 w-[150px] h-[200px] border border-white/10 rounded-3xl p-4 text-sm text-neutral-800 dark:text-neutral-300 cursor-pointer hover:scale-105 transition-all"
                                            onClick={() => handleSelectExample(example.description)}>
                                            {example.icon}
                                            {example.description}
                                        </div>
                                    ))
                                }
                            </div>
                        </div>
                        : <div className="h-full w-full py-8 max-h-[calc(80vh-5px)] overflow-y-scroll" ref={scrollContainerRef}>
                            <div className="flex justify-center w-full">
                                <div className="flex w-[max(65%,450px)] flex-col gap-2">
                                    {
                                        chats.map((chat, i) => {
                                            if (chat.Type === "message" || chat.Type === "text") {
                                                return <div key={`chat-${i}`} className={classNames("flex gap-4 overflow-hidden break-words leading-6 shrink-0 relative", {
                                                    "self-end": chat.isUserInput,
                                                    "self-start": !chat.isUserInput,
                                                })}>
                                                    {!chat.isUserInput && chats[i-1]?.isUserInput
                                                        ? <img src={logoImage} alt="clidey logo" className="w-auto h-6 mt-2" />
                                                        : <div className="pl-4" />}
                                                    <div className={classNames("text-neutral-800 dark:text-neutral-300 px-4 py-2 rounded-xl whitespace-pre-wrap", {
                                                        "bg-neutral-600/5 dark:bg-[#2C2F33]": chat.isUserInput,
                                                        "-ml-2": !chat.isUserInput && chats[i-1]?.isUserInput,
                                                    })}>
                                                        {chat.Text}
                                                    </div>
                                                </div>
                                            } else if (chat.Type === "error") {
                                                return <div key={`chat-${i}`} className="flex items-center gap-4 overflow-hidden break-words leading-6 shrink-0 self-start">
                                                    {!chat.isUserInput && chats[i-1]?.isUserInput
                                                        ? <img src={logoImage} alt="clidey logo" className="w-auto h-6" />
                                                        : <div className="pl-4" />}
                                                    <div className="text-red-800 dark:text-red-300 px-4 py-2 rounded-lg">
                                                        {chat.Text}
                                                    </div>
                                                </div>
                                            } else if (isEEFeatureEnabled('dataVisualization') && (chat.Type === "sql:pie-chart" || chat.Type === "sql:line-chart")) {
                                                return <div key={`chat-${i}`} className="flex items-center self-start">
                                                    {!chat.isUserInput && chats[i-1]?.isUserInput && <img src={logoImage} alt="clidey logo" className="w-auto h-6" />}
                                                    {/* @ts-ignore */}
                                                    {chat.Type === "sql:pie-chart" && PieChart && <PieChart columns={chat.Result?.Columns.map(col => col.Name) ?? []} data={chat.Result?.Rows ?? []} />}
                                                    {/* @ts-ignore */}
                                                    {chat.Type === "sql:line-chart" && LineChart && <LineChart columns={chat.Result?.Columns.map(col => col.Name) ?? []} data={chat.Result?.Rows ?? []} />}
                                                </div>
                                            }
                                            return <div key={`chat-${i}`} className="flex gap-4 w-full overflow-hidden pt-4 pr-9">
                                                {!chat.isUserInput && chats[i-1]?.isUserInput
                                                    ? <img src={logoImage} alt="clidey logo" className="w-auto h-6" />
                                                    : <div className="pl-4" />}
                                                <TablePreview type={chat.Type} text={chat.Text} data={chat.Result} />
                                            </div>
                                        })
                                    }
                                    { loading &&  <div className="flex w-full mt-4">
                                        <Loading loadingText={isEEMode ? thinkingPhrases[0] : chooseRandomItems(thinkingPhrases)[0]} size="sm" />
                                    </div> }
                                </div>
                            </div>
                        </div>
                    }
                </div>
                <div className={classNames("relative w-full", {
                    "opacity-80": disableChat,
                    "opacity-10": disableAll,
                })}>
                    <div className={classNames("absolute right-2 top-1/2 -translate-y-1/2 z-10 backdrop-blur-lg rounded-full cursor-pointer hover:scale-110 transition-all", {
                        "opacity-50": loading,
                    })} onClick={loading ? undefined : handleSubmitQuery}>
                        {cloneElement(Icons.ArrowUp, {
                            className: "w-5 h-5 stroke-neutral-800 dark:stroke-neutral-300",
                        })}
                    </div>
                    <Input value={query} setValue={setQuery} placeholder="Talk to me..." onSubmit={handleSubmitQuery} inputProps={{
                        disabled: disableChat,
                        onKeyUp: handleKeyUp,
                        ref: (input) => input?.focus()
                    }} />
                </div>
                {
                    (!modelAvailable || models.length === 0) &&
                    <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 text-neutral-800 dark:text-neutral-300">
                        Please choose an available model
                    </div>
                }
            </div>
        </InternalPage>
    )
}
