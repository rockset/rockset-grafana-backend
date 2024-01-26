import React, { useState } from 'react';
import { RocksetVariableQuery } from '../types';

interface VariableQueryProps {
    query: RocksetVariableQuery;
    onChange: (query: RocksetVariableQuery, definition: string) => void;
}

export const VariableQueryEditor: React.FC<VariableQueryProps> = ({ onChange, query }) => {
    const [state, setState] = useState(query);

    const saveQuery = () => {
        onChange(state, `${state.rawQuery}`);
    };

    const handleChange = (event: React.FormEvent<HTMLInputElement>) =>
        setState({
            ...state,
            [event.currentTarget.name]: event.currentTarget.value,
        });

    return (
        <>
            <div className="gf-form">
                <span className="gf-form-label width-10">Query</span>
                <input
                    name="rawQuery"
                    className="gf-form-input"
                    onBlur={saveQuery}
                    onChange={handleChange}
                    value={state.rawQuery}
                />
            </div>
        </>
    );
};
